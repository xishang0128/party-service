package manager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sparkle-service/manager/sandbox"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigCheck 测试配置
func ConfigCheck(path string) error {
	if path == "" {
		return fmt.Errorf("配置文件路径不能为空")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	s, p, g := randThreeStrings(10)
	p1, p2, config, err := parseConfig(data, s, p, g)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	proc, err := startProcess(config)
	if err != nil {
		return fmt.Errorf("进程启动失败: %s", err)
	}

	if err := runTests(proc, p1, p2, s, p, g); err != nil {
		return fmt.Errorf("测试失败: %v", err)
	}

	proc.Stop()
	return nil
}

func startProcess(config []byte) (*sandbox.SandboxedProcess, error) {
	proc, err := sandbox.NewSandboxedProcess(sandbox.Config{
		BinaryPath: "/home/atri/mihomo/mihomo",
		WorkDir:    "/home/atri/mihomo",
		Args:       []string{"-config", base64.StdEncoding.EncodeToString(config)},
	})
	if err != nil {
		return nil, err
	}

	if err := proc.Start(); err != nil {
		return nil, err
	}

	return proc, nil
}

func runTests(proc *sandbox.SandboxedProcess, proxyPort, controllerPort int, secret, proxie, group string) error {
	if err := checkProxy(proc.StdoutBuffer(), proxyPort); err != nil {
		return err
	}

	if _, err := makeRequest(controllerPort, secret, http.MethodGet, "", nil); err != nil {
		return fmt.Errorf("控制器检查失败: %v", err)
	}

	if _, err := makeRequest(controllerPort, secret, http.MethodPut, "/proxies/group", map[string]any{
		"name": "http",
	}); err != nil {
		return fmt.Errorf("切换代理失败: %v", err)
	}

	resp, err := makeRequest(controllerPort, secret, http.MethodGet, "/proxies/group", nil)
	if err != nil {
		return fmt.Errorf("获取代理失败: %v", err)
	}
	if !strings.ContainsAny(resp, proxie) && !strings.ContainsAny(resp, group) {
		return fmt.Errorf("获取代理失败")
	}

	return nil
}

func parseConfig(data []byte, secret, proxie, group string) (int, int, []byte, error) {
	var conf map[string]any
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return 0, 0, nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	if tun, ok := conf["tun"].(map[string]any); ok {
		tun["enable"] = false
	}

	p1, p2, err := findAvailablePorts()
	if err != nil {
		return 0, 0, nil, err
	}

	conf["secret"] = secret
	conf["external-controller"] = fmt.Sprintf("127.0.0.1:%d", p2)
	conf["log-level"] = "info"
	conf["mode"] = "rule"
	listeners := get(conf, "listeners")
	conf["listeners"] = append(listeners, map[string]any{
		"type":   "mixed",
		"port":   p1,
		"listen": "127.0.0.1",
	})
	proxies := get(conf, "proxies")
	conf["proxies"] = append(proxies, map[string]any{
		"name":   proxie,
		"type":   "http",
		"server": "127.0.0.1",
		"port":   1080,
	})
	groups := get(conf, "proxy-groups")
	conf["proxy-groups"] = append(groups, map[string]any{
		"name":    group,
		"type":    "select",
		"proxies": []string{"DIRECT", proxie},
	})

	config, err := yaml.Marshal(&conf)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("序列化配置文件失败: %v", err)
	}
	return p1, p2, config, nil
}

func get(conf map[string]any, name string) []any {
	if listeners, ok := conf[name].([]any); ok {
		return listeners
	}
	return []any{}
}

func findAvailablePorts() (int, int, error) {
	l1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "端口获取失败: %v\n", err)
		os.Exit(1)
	}
	defer l1.Close()

	l2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, 0, fmt.Errorf("端口获取失败: %v", err)
	}
	defer l2.Close()

	return l1.Addr().(*net.TCPAddr).Port, l2.Addr().(*net.TCPAddr).Port, nil
}

func randThreeStrings(n int) (string, string, string) {
	return randString(n), randString(n), randString(n)
}

func randString(n int) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func checkProxy(outBuffer *bytes.Buffer, port int) error {
	deadline := time.Now().Add(startTimeout)

	for time.Now().Before(deadline) {
		output := outBuffer.String()
		if strings.Contains(output, successIndicator) {
			proxy, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
			client := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxy),
				},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			if _, err := client.Get("http://1.1.1.1"); err == nil && strings.Contains(outBuffer.String(), "1.1.1.1:80") {
				return nil
			}
			return fmt.Errorf("代理测试失败")
		}
		if strings.Contains(output, fatalIndicator) {
			if msgStart := strings.Index(output, "level=fatal msg="); msgStart != -1 {
				return fmt.Errorf("配置错误: %s", output[msgStart+16:])
			}
			return fmt.Errorf("发生致命错误")
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("启动超时")
}

func makeRequest(port int, token string, method string, path string, data any) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("JSON编码失败: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	return string(respBody), nil
}
