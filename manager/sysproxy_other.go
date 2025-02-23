//go:build !darwin && !linux && !windows

package manager

import "fmt"

func DisableProxy() error {
	return fmt.Errorf("不支持的操作系统")
}

func SetProxy(proxy, bypass string) error {
	return fmt.Errorf("不支持的操作系统")
}

func SetPac(server string) error {
	return fmt.Errorf("不支持的操作系统")
}

func QueryProxySettings() (map[string]any, error) {
	return nil, fmt.Errorf("不支持的操作系统")
}
