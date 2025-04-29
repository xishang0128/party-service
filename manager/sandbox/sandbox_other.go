//go:build !linux && !windows && !darwin

package sandbox

import (
	"fmt"
	"os/exec"
	"runtime"
)

func applySandboxLimits(_ *exec.Cmd) error {
	return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
}

func startCmdInSandbox(_ *exec.Cmd) error {
	return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
}
