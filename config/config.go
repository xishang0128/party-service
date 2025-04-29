package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	defaultEncryptionKey = "s5PYArGj8RdLC6rKZfxQttlMFt17lBlY"
	defaultConfigFile    = "sparkle-config.yaml"
)

type ConfigManager struct {
	sync.RWMutex
	cfg        *Config
	configFile string
	encryptKey []byte
}

type Config struct {
	CoreName   EncryptedString `yaml:"core-name"`
	ConfigPath EncryptedString `yaml:"config-path"`
	CoreDir    EncryptedString `yaml:"core-dir"`
	WorkDir    EncryptedString `yaml:"workdir"`
	LogPath    EncryptedString `yaml:"log-path"`
	Secret     EncryptedString `yaml:"secret"`
	Http       EncryptedString `yaml:"http-controller"`
	NamedPipe  EncryptedString `yaml:"named-pipe"`
	UnixSocket EncryptedString `yaml:"unix-socket"`
}

type EncryptedString string

var (
	manager *ConfigManager
	once    sync.Once
)

func Initialize(configFile string, encryptKey string) error {
	var err error
	once.Do(func() {
		if configFile == "" {
			configFile = defaultConfigFile
		}
		if encryptKey == "" {
			encryptKey = defaultEncryptionKey
		}

		manager = &ConfigManager{
			configFile: configFile,
			encryptKey: []byte(encryptKey),
			cfg:        &Config{},
		}

		err = manager.load()
	})
	return err
}

func (cm *ConfigManager) load() error {
	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cm.save()
		}
		return fmt.Errorf("读取配置文件失败：%w", err)
	}

	if err := yaml.Unmarshal(data, cm.cfg); err != nil {
		return fmt.Errorf("解析配置文件失败：%w", err)
	}
	return nil
}

func (cm *ConfigManager) save() error {
	cm.Lock()
	defer cm.Unlock()

	out, err := yaml.Marshal(cm.cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败：%w", err)
	}

	if err := os.WriteFile(cm.configFile, out, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败：%w", err)
	}
	return nil
}

func (cm *ConfigManager) getString(value EncryptedString) string {
	cm.RLock()
	defer cm.RUnlock()
	return string(value)
}

func (cm *ConfigManager) setString(dest *EncryptedString, value string) error {
	cm.Lock()
	if value != "" {
		*dest = EncryptedString(value)
	}
	cm.Unlock()
	return cm.save()
}

func GetConfig() Config {
	return Config{
		CoreName:   EncryptedString(GetCoreName()),
		CoreDir:    EncryptedString(GetCoreDir()),
		ConfigPath: EncryptedString(GetConfigPath()),
		WorkDir:    EncryptedString(GetWorkDir()),
		LogPath:    EncryptedString(GetLogPath()),
		Secret:     EncryptedString(GetSecret()),
		Http:       EncryptedString(GetHttp()),
		NamedPipe:  EncryptedString(GetNamedPipe()),
		UnixSocket: EncryptedString(GetUnixSocket()),
	}
}

func UpdateConfig(coreName, coreDir, configPath, workDir, logPath, secret, http, namedPipe, unixSocket string) error {
	if err := manager.setString(&manager.cfg.CoreName, coreName); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.CoreDir, coreDir); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.ConfigPath, configPath); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.WorkDir, workDir); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.LogPath, logPath); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.Secret, secret); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.Http, http); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.NamedPipe, namedPipe); err != nil {
		return err
	}
	if err := manager.setString(&manager.cfg.UnixSocket, unixSocket); err != nil {
		return err
	}

	return nil
}

func GetCoreName() string   { return manager.getString(manager.cfg.CoreName) }
func GetCoreDir() string    { return manager.getString(manager.cfg.CoreDir) }
func GetConfigPath() string { return manager.getString(manager.cfg.ConfigPath) }
func GetWorkDir() string    { return manager.getString(manager.cfg.WorkDir) }
func GetLogPath() string    { return manager.getString(manager.cfg.LogPath) }
func GetSecret() string     { return manager.getString(manager.cfg.Secret) }
func GetHttp() string       { return manager.getString(manager.cfg.Http) }
func GetNamedPipe() string  { return manager.getString(manager.cfg.NamedPipe) }
func GetUnixSocket() string { return manager.getString(manager.cfg.UnixSocket) }

func (es EncryptedString) MarshalYAML() (any, error) {
	block, err := aes.NewCipher(manager.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("创建加密器失败：%w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 加密器失败：%w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成随机数失败：%w", err)
	}

	ct := gcm.Seal(nonce, nonce, []byte(es), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (es *EncryptedString) UnmarshalYAML(value *yaml.Node) error {
	var cipherB64 string
	if err := value.Decode(&cipherB64); err != nil {
		return fmt.Errorf("解码 base64 失败：%w", err)
	}

	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return fmt.Errorf("解码密文失败：%w", err)
	}

	block, err := aes.NewCipher(manager.encryptKey)
	if err != nil {
		return fmt.Errorf("创建解密器失败：%w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("创建 GCM 解密器失败：%w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return fmt.Errorf("密文长度不足")
	}

	nonce, ct := data[:nonceSize], data[nonceSize:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return fmt.Errorf("解密失败：%w", err)
	}

	*es = EncryptedString(pt)
	return nil
}
