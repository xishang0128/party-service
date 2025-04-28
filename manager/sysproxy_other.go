//go:build !darwin && !linux && !windows

package manager

import "fmt"

func DisableProxy(_ string, _ uint32) error {
	return fmt.Errorf("不支持的操作系统")
}

func SetProxy(_, _, _ string, _ uint32) error {
	return fmt.Errorf("不支持的操作系统")
}

func SetPac(_, _ string, _ uint32) error {
	return fmt.Errorf("不支持的操作系统")
}

func QueryProxySettings(_ string, _ uint32) (map[string]any, error) {
	return nil, fmt.Errorf("不支持的操作系统")
}
