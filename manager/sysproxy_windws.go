//go:build windows

package manager

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	INTERNET_OPTION_REFRESH                = 37
	INTERNET_OPTION_PROXY_SETTINGS_CHANGED = 39
	INTERNET_OPTION_PER_CONNECTION_OPTION  = 75

	INTERNET_PER_CONN_FLAGS          = 1
	INTERNET_PER_CONN_PROXY_SERVER   = 2
	INTERNET_PER_CONN_PROXY_BYPASS   = 3
	INTERNET_PER_CONN_AUTOCONFIG_URL = 4

	PROXY_TYPE_DIRECT         = 1
	PROXY_TYPE_PROXY          = 2
	PROXY_TYPE_AUTO_PROXY_URL = 4
)

var (
	wininet                  = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOptionW   = wininet.NewProc("InternetSetOptionW")
	procInternetQueryOptionW = wininet.NewProc("InternetQueryOptionW")
)

type (
	InternetPerConnOption struct {
		dwOption uint32
		dwValue  uintptr
	}

	InternetPerConnOptionList struct {
		dwSize        uint32
		pszConnection *uint16
		dwOptionCount uint32
		dwOptionError uint32
		pOptions      *InternetPerConnOption
	}
)

func refreshAndApplySettings(options []InternetPerConnOption) error {
	list := InternetPerConnOptionList{
		dwSize:        uint32(unsafe.Sizeof(InternetPerConnOptionList{})),
		dwOptionCount: uint32(len(options)),
		pOptions:      &options[0],
	}

	if ret, _, err := procInternetSetOptionW.Call(
		0,
		INTERNET_OPTION_PER_CONNECTION_OPTION,
		uintptr(unsafe.Pointer(&list)),
		unsafe.Sizeof(list)); ret == 0 {
		return fmt.Errorf("set option failed: %v", err)
	}

	procInternetSetOptionW.Call(0, INTERNET_OPTION_PROXY_SETTINGS_CHANGED, 0, 0)
	procInternetSetOptionW.Call(0, INTERNET_OPTION_REFRESH, 0, 0)
	return nil
}

func DisableProxy() error {
	return refreshAndApplySettings([]InternetPerConnOption{{
		dwOption: INTERNET_PER_CONN_FLAGS,
		dwValue:  PROXY_TYPE_DIRECT,
	}})
}

func SetProxy(proxy, bypass string) error {
	if proxy == "" || bypass == "" {
		settings, err := QueryProxySettings()
		if err != nil {
			return err
		}

		proxySettings := settings["proxy"].(map[string]any)
		if proxy == "" {
			proxy = proxySettings["server"].(string)
		}
		if bypass == "" {
			bypass = proxySettings["bypass"].(string)
		}
	}
	proxyPtr, err := syscall.UTF16PtrFromString(proxy)
	if err != nil {
		return err
	}
	bypassPtr, err := syscall.UTF16PtrFromString(bypass)
	if err != nil {
		return err
	}

	return refreshAndApplySettings([]InternetPerConnOption{
		{dwOption: INTERNET_PER_CONN_FLAGS, dwValue: PROXY_TYPE_PROXY},
		{dwOption: INTERNET_PER_CONN_PROXY_SERVER, dwValue: uintptr(unsafe.Pointer(proxyPtr))},
		{dwOption: INTERNET_PER_CONN_PROXY_BYPASS, dwValue: uintptr(unsafe.Pointer(bypassPtr))},
	})
}

func SetPac(pacUrl string) error {
	if pacUrl == "" {
		return refreshAndApplySettings([]InternetPerConnOption{
			{dwOption: INTERNET_PER_CONN_FLAGS, dwValue: PROXY_TYPE_AUTO_PROXY_URL},
		})
	}
	pacPtr, err := syscall.UTF16PtrFromString(pacUrl)
	if err != nil {
		return err
	}

	return refreshAndApplySettings([]InternetPerConnOption{
		{dwOption: INTERNET_PER_CONN_FLAGS, dwValue: PROXY_TYPE_AUTO_PROXY_URL},
		{dwOption: INTERNET_PER_CONN_AUTOCONFIG_URL, dwValue: uintptr(unsafe.Pointer(pacPtr))},
	})
}

func QueryProxySettings() (map[string]any, error) {
	options := [4]InternetPerConnOption{
		{dwOption: INTERNET_PER_CONN_FLAGS},
		{dwOption: INTERNET_PER_CONN_PROXY_SERVER},
		{dwOption: INTERNET_PER_CONN_PROXY_BYPASS},
		{dwOption: INTERNET_PER_CONN_AUTOCONFIG_URL},
	}

	list := InternetPerConnOptionList{
		dwSize:        uint32(unsafe.Sizeof(InternetPerConnOptionList{})),
		dwOptionCount: 4,
		pOptions:      &options[0],
	}

	if ret, _, err := procInternetQueryOptionW.Call(
		0,
		INTERNET_OPTION_PER_CONNECTION_OPTION,
		uintptr(unsafe.Pointer(&list)),
		uintptr(unsafe.Pointer(&list.dwSize))); ret == 0 {
		return nil, fmt.Errorf("query failed: %v", err)
	}

	flags := uint32(options[0].dwValue)
	result := map[string]any{
		"flags": flags,
		"proxy": map[string]any{
			"enable": (flags & PROXY_TYPE_PROXY) != 0,
			"server": getString(options[1].dwValue),
			"bypass": getString(options[2].dwValue),
		},
		"pac": map[string]any{
			"enable": (flags & PROXY_TYPE_AUTO_PROXY_URL) != 0,
			"url":    getString(options[3].dwValue),
		},
	}

	return result, nil
}

func getString(val uintptr) string {
	if val == 0 {
		return ""
	}
	return syscall.UTF16ToString(*(*[]uint16)(unsafe.Pointer(&struct {
		addr uintptr
		len  int
		cap  int
	}{val, 1024, 1024})))
}
