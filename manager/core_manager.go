package manager

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sparkle-service/config"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

const (
	startTimeout     = 30 * time.Second
	checkInterval    = 500 * time.Millisecond
	successIndicator = "Start initial Compatible provider default"
	fatalIndicator   = "level=fatal"
	monitorInterval  = 1 * time.Second
)

type CoreManager struct {
	cmd        *exec.Cmd
	isRunning  atomic.Bool
	monitoring atomic.Bool
	startTime  time.Time
	pid        atomic.Int32
	mutex      sync.Mutex
	stopChan   chan struct{}
}

type ProcessInfo struct {
	PID          int32     `json:"pid"`
	Memory       uint64    `json:"memory"`
	MemoryFormat string    `json:"memory_format"`
	StartTime    time.Time `json:"start_time"`
	Uptime       string    `json:"uptime"`
}

func NewCoreManager() *CoreManager {
	return &CoreManager{
		stopChan: make(chan struct{}),
	}
}

func (cm *CoreManager) getCorePath() string {
	return filepath.Join(config.GetCoreDir(), config.GetCoreName())
}

func (cm *CoreManager) StartCore() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if !cm.isRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("核心进程已在运行中")
	}

	cm.stopChan = make(chan struct{})

	if !cm.monitoring.Load() {
		cm.monitoring.Store(true)
		go cm.monitorPID()
	}

	return cm.startProcess()
}

func (cm *CoreManager) startProcess() error {
	var configPath string
	if config.GetConfigPath() == "" {
		configPath = filepath.Join(config.GetWorkDir(), "config.yaml")
	} else {
		configPath = config.GetConfigPath()
	}

	if err := ConfigTest(configPath); err != nil {
		cm.isRunning.Store(false)
		return fmt.Errorf("配置测试失败: %w", err)
	}

	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	multiWriter := io.MultiWriter(os.Stdout, outBuffer)

	cmd := cm.buildCommand()
	cmd.Stdout = multiWriter
	cmd.Stderr = errBuffer
	cmd.Env = append(os.Environ(), "DISABLE_LOOPBACK_DETECTOR=true")

	if err := cmd.Start(); err != nil {
		cm.isRunning.Store(false)
		return fmt.Errorf("启动核心进程失败: %w", err)
	}

	cm.cmd = cmd
	cm.pid.Store(int32(cmd.Process.Pid))
	cm.startTime = time.Now()

	go cm.monitorProcess(errBuffer)

	return cm.waitForStartup(outBuffer)
}

func (cm *CoreManager) StopCore() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if !cm.isRunning.Load() {
		return nil
	}

	close(cm.stopChan)
	cm.monitoring.Store(false)

	if err := cm.stopProcess(); err != nil {
		return err
	}

	cm.cleanup()
	return nil
}

func (cm *CoreManager) stopProcess() error {
	processName := filepath.Base(cm.getCorePath())
	if runtime.GOOS == "windows" {
		processName += ".exe"
		return cm.killProcessWindows(processName)
	}
	return cm.killProcessUnix(processName)
}

func (cm *CoreManager) cleanup() {
	cm.isRunning.Store(false)
	cm.cmd = nil
	cm.pid.Store(0)
}

func (cm *CoreManager) RestartCore() error {
	cm.monitoring.Store(false)
	if err := cm.StopCore(); err != nil {
		log.Printf("停止进程时出错: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	return cm.StartCore()
}

func (cm *CoreManager) buildCommand() *exec.Cmd {
	corePath := cm.getCorePath()
	if runtime.GOOS == "windows" {
		corePath = cm.getCorePath() + ".exe"
	}
	cmd := exec.Command(
		corePath,
		"-d", config.GetWorkDir(),
	)
	if config.GetConfigPath() != "" {
		cmd.Args = append(cmd.Args, "-f", config.GetConfigPath())
	}
	if config.GetHttp() != "" {
		cmd.Args = append(cmd.Args, "-ext-ctl", config.GetHttp())
	}
	if config.GetUnixSocket() != "" {
		cmd.Args = append(cmd.Args, "-ext-ctl-unix", config.GetUnixSocket())
	}
	if config.GetNamedPipe() != "" && runtime.GOOS == "windows" {
		cmd.Args = append(cmd.Args, "-ext-ctl-pipe", config.GetNamedPipe())
	}
	if config.GetSecret() != "" {
		cmd.Args = append(cmd.Args, "-secret", config.GetSecret())
	}
	return cmd
}

func (cm *CoreManager) monitorProcess(errBuffer *bytes.Buffer) {
	if err := cm.cmd.Wait(); err != nil {
		if cm.monitoring.Load() {
			log.Printf("核心进程异常退出: %v\n错误输出: %s", err, errBuffer.String())
			cm.handleProcessExit()
		}
	}
}

func (cm *CoreManager) handleProcessExit() {
	cm.isRunning.Store(false)
	if cm.monitoring.Load() {
		go func() {
			for retries := range 3 {
				if err := cm.RestartCore(); err != nil {
					log.Printf("重启核心进程失败 (尝试 %d/3): %v", retries+1, err)
					time.Sleep(time.Second * time.Duration(retries+1))
					continue
				}
				log.Println("核心进程已成功重启")
				return
			}
			log.Println("达到最大重试次数，重启失败")
		}()
	}
}

func (cm *CoreManager) monitorPID() {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	processName := filepath.Base(cm.getCorePath())
	if runtime.GOOS == "windows" {
		processName += ".exe"
	}

	for {
		select {
		case <-ticker.C:
			if !cm.monitoring.Load() {
				return
			}

			exists, newPID := cm.checkProcess(processName)
			if !exists {
				if cm.isRunning.Load() {
					log.Printf("核心进程已终止 (PID: %d)", cm.pid.Load())
					cm.handleProcessExit()
				}
				continue
			}

			if newPID != cm.pid.Load() {
				log.Printf("检测到核心进程PID变化 (PID: %d -> %d)", cm.pid.Load(), newPID)
				cm.updatePID(newPID)
			}
		case <-cm.stopChan:
			return
		}
	}
}

func (cm *CoreManager) checkProcess(processName string) (bool, int32) {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("获取进程列表失败: %v", err)
		return false, 0
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if name == processName {
			return true, p.Pid
		}
	}
	return false, 0
}

func (cm *CoreManager) updatePID(newPID int32) {
	cm.pid.Store(newPID)
	if proc, err := process.NewProcess(newPID); err == nil {
		cm.updateProcessInfo(proc)
	}
}

// updateProcessInfo 更新进程信息
func (cm *CoreManager) updateProcessInfo(p *process.Process) {
	cm.cmd = &exec.Cmd{
		Process: &os.Process{Pid: int(p.Pid)},
	}
	cm.pid.Store(p.Pid)

	if createTime, err := p.CreateTime(); err == nil {
		cm.startTime = time.UnixMilli(createTime)
	}
}

// waitForStartup 等待启动完成
func (cm *CoreManager) waitForStartup(outBuffer *bytes.Buffer) error {
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			output := outBuffer.String()
			if strings.Contains(output, successIndicator) {
				return nil
			}
			if strings.Contains(output, fatalIndicator) {
				return cm.extractFatalError(output)
			}
		case <-ctx.Done():
			cm.isRunning.Store(false)
			return fmt.Errorf("启动核心进程超时")
		}
	}
}

func (cm *CoreManager) IsHealthy() bool {
	if !cm.isRunning.Load() {
		return false
	}

	info, err := cm.GetProcessInfo()
	if err != nil {
		return false
	}

	if info.Memory > 1024*1024*1024 { // 1GB
		log.Printf("警告: 核心进程内存使用过高 (%s)", info.MemoryFormat)
	}

	return true
}

// GetProcessInfo 获取进程信息
func (cm *CoreManager) GetProcessInfo() (*ProcessInfo, error) {
	if !cm.isRunning.Load() || cm.cmd == nil || cm.cmd.Process == nil {
		return nil, fmt.Errorf("进程未运行")
	}

	currentPID := cm.pid.Load()
	proc, err := process.NewProcess(currentPID)
	if err != nil {
		return nil, fmt.Errorf("获取进程信息失败：%w", err)
	}

	info := &ProcessInfo{
		PID:       currentPID,
		StartTime: cm.startTime,
		Uptime:    formatUptime(time.Since(cm.startTime)),
	}

	if memInfo, err := proc.MemoryInfo(); err == nil {
		info.Memory = memInfo.RSS
		info.MemoryFormat = formatMemory(memInfo.RSS)
	}

	return info, nil
}

// 以下是辅助函数
func formatMemory(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := make([]string, 0, 4)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))

	return strings.Join(parts, " ")
}

// 其他辅助函数保持不变...
// ConfigTest, parseConfig, getListeners, findAvailablePort, copyDir, copyFile 等函数代码保持原样

func (cm *CoreManager) convertGBKToUTF8(b []byte) string {
	reader := transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewDecoder())
	output, err := io.ReadAll(reader)
	if err != nil {
		return string(b)
	}
	return string(output)
}

func (cm *CoreManager) extractFatalError(output string) error {
	if msgStart := strings.Index(output, "level=fatal msg="); msgStart != -1 {
		msg := strings.TrimSpace(output[msgStart+16:])
		return fmt.Errorf("启动核心进程失败: %s", msg)
	}
	return fmt.Errorf("启动核心进程失败：发现致命错误")
}

func (cm *CoreManager) killProcessWindows(processName string) error {
	cmd := exec.Command("taskkill", "/F", "/IM", processName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := cm.convertGBKToUTF8(output)
		if strings.Contains(outputStr, "没有找到进程") {
			return nil
		}
		return fmt.Errorf("终止进程失败: %v, output: %s", err, outputStr)
	}

	maxRetries := 5
	for range maxRetries {
		if !cm.isProcessRunning(processName) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (cm *CoreManager) isProcessRunning(processName string) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/NH")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), processName)
}

func (cm *CoreManager) killProcessUnix(processName string) error {
	output, err := exec.Command("pkill", "-f", processName).CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return fmt.Errorf("终止进程失败: %w, output: %s", err, string(output))
	}

	cm.isRunning.Store(false)
	log.Printf("成功终止进程 %s", processName)
	return nil
}

// ConfigTest 测试配置
func ConfigTest(path string) error {
	port, conf, err := parseConfig(path)
	if err != nil {
		return fmt.Errorf("配置解析失败: %v", err)
	}

	tmpPath := config.GetWorkDir() + "/tmp"
	err = os.MkdirAll(tmpPath, 0755)
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	err = copyDir(config.GetWorkDir(), tmpPath)
	if err != nil {
		return fmt.Errorf("复制工作目录失败: %v", err)
	}
	defer os.RemoveAll(tmpPath)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command(
			config.GetCoreDir()+"/"+config.GetCoreName()+".exe",
			"-d", tmpPath,
			"--config", base64.StdEncoding.EncodeToString(conf),
		)
	} else {
		cmd = exec.Command(
			config.GetCoreDir()+"/"+config.GetCoreName(),
			"-d", tmpPath,
			"--config", base64.StdEncoding.EncodeToString(conf),
		)
	}
	defer func() {
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		os.RemoveAll(tmpPath)
	}()

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动核心进程失败: %v", err)
	}
	defer cmd.Process.Kill()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			output := outBuffer.String()
			if strings.Contains(output, successIndicator) {
				proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
				if err != nil {
					return fmt.Errorf("代理地址解析失败: %v", err)
				}
				client := &http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxyURL),
					},
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				client.Get("http://1.1.1.1")
				if strings.Contains(outBuffer.String(), "1.1.1.1:80") {
					return nil
				}
				return fmt.Errorf("测试失败")
			}

			if strings.Contains(output, fatalIndicator) {
				if msgStart := strings.Index(output, "level=fatal msg="); msgStart != -1 {
					return fmt.Errorf("配置错误: %s", output[msgStart+16:])
				}
				return fmt.Errorf("发生致命错误")
			}

		case <-ctx.Done():
			return fmt.Errorf("配置测试超时")
		}
	}
}

func parseConfig(path string) (int, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var conf map[string]any
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return 0, nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	if tun, ok := conf["tun"].(map[string]any); ok {
		tun["enable"] = false
	}

	port := findAvailablePort()
	listeners := getListeners(conf)
	listeners = append(listeners, map[string]any{
		"type":   "mixed",
		"port":   port,
		"listen": "127.0.0.1",
	})
	conf["listeners"] = listeners
	conf["log-level"] = "info"

	config, err := yaml.Marshal(&conf)
	if err != nil {
		return 0, nil, fmt.Errorf("序列化配置文件失败: %v", err)
	}
	return port, config, nil
}

func getListeners(conf map[string]any) []any {
	if listeners, ok := conf["listeners"].([]any); ok {
		return listeners
	}
	return []any{}
}

func findAvailablePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println("未能找到可用端口")
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
