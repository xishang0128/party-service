package manager

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	cmd        *exec.Cmd
	corePath   = "D:\\Mihomo Party\\resources\\sidecar\\mihomo-alpha.exe"
	workDir    = "C:\\Users\\atri\\AppData\\Roaming\\mihomo-party\\work"
	configPath = "C:\\Users\\atri\\AppData\\Roaming\\mihomo-party\\work\\config.yaml"

	logFile = "process.log"

	config []byte

	httpPort int
)

func StartCore() error {
	args := []string{"-d", workDir, "-ext-ctl-unix", "D:\\git\\xishang0128\\party-service\\party.sock"}
	env := os.Environ()

	var outBuffer, errBuffer bytes.Buffer
	multiWriter := io.MultiWriter(os.Stdout, &outBuffer)

	cmd = exec.Command(corePath, args...)
	cmd.Env = append(env, "DISABLE_LOOPBACK_DETECTOR=true")
	cmd.Stdout = multiWriter
	cmd.Stderr = &errBuffer

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动核心进程失败")
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("核心进程退出")
		}
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		output := outBuffer.String()
		if strings.Contains(output, "Start initial Compatible provider GLOBAL") {
			return nil
		}
		if strings.Contains(output, "level=fatal") {
			msgStart := strings.Index(output, "msg=")
			if msgStart != -1 {
				msg := output[msgStart+4:]
				msg = strings.TrimSpace(msg)
				return fmt.Errorf("启动核心进程失败: %s", msg)
			}
		}
	}
	return fmt.Errorf("启动核心进程失败")
}

func StopCore() error {
	if cmd != nil && cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("终止核心进程失败: %v", err)
		} else {
			log.Println("核心进程已终止")
		}
	}
	return nil
}

func RestartCore() error {
	if err := StopCore(); err != nil {
		return err
	}
	return StartCore()
}

func ConfigTest() error {
	if err := configParse(); err != nil {
		return fmt.Errorf("配置解析失败: %v", err)
	}

	cmd := exec.Command(corePath, "-d", workDir, "--config", base64.StdEncoding.EncodeToString(config))

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动核心进程失败: %v", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		output := outBuffer.String()
		if strings.Contains(output, "Start initial Compatible provider GLOBAL") {
			if proxyTest() == nil && strings.Contains(outBuffer.String(), "1.1.1.1:80") {
				cmd.Process.Kill()
				return nil
			}
			return fmt.Errorf("测试失败")
		}

		if strings.Contains(output, "level=fatal") {
			if msgStart := strings.Index(output, "level=fatal msg="); msgStart != -1 {
				return fmt.Errorf("配置错误: %s", output[msgStart+16:])
			}
		}
	}

	return fmt.Errorf("超时未检测到初始化完成")
}

func proxyTest() error {
	proxyURL, err := url.Parse("http://127.0.0.1:" + fmt.Sprint(httpPort))
	if err != nil {
		return fmt.Errorf("解析代理URL失败: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client.Get("http://1.1.1.1")

	return nil
}

func configParse() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取 YAML 文件失败: %v", err)
	}

	var conf map[string]any
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return fmt.Errorf("解析 YAML 文件失败: %v", err)
	}

	if tun, ok := conf["tun"].(map[string]any); ok {
		tun["enable"] = false
	}

	findAvailablePort()
	listeners, ok := conf["listeners"].([]any)
	if !ok {
		listeners = []any{}
	}
	newListener := map[string]any{
		"type":   "mixed",
		"port":   httpPort,
		"listen": "127.0.0.1",
	}
	listeners = append(listeners, newListener)
	conf["listeners"] = listeners

	config, err = yaml.Marshal(&conf)
	if err != nil {
		return fmt.Errorf("序列化 YAML 文件失败: %v", err)
	}

	return nil
}

func findAvailablePort() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("未能找到可用端口: %v", err)
	}
	defer listener.Close()

	httpPort = listener.Addr().(*net.TCPAddr).Port
}

func LogToFile(message string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("打开日志文件失败: %v\n", err)
	}
	defer file.Close()
	file.WriteString(message + "\n")
}
