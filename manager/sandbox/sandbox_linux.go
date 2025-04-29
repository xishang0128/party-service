//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func applySandboxLimits(cmd *exec.Cmd) error {
	uid := os.Getuid()
	gid := os.Getgid()

	chrootDir := cmd.Dir
	cmd.Dir = "/"

	binName := filepath.Base(cmd.Path)
	cmd.Path = filepath.Join("/", binName)

	hostBinPath := filepath.Join(chrootDir, binName)
	if err := os.Chmod(hostBinPath, 0755); err != nil {
		return fmt.Errorf("设置执行权限失败: %w", err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS,

		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		},
		GidMappingsEnableSetgroups: false,
		Chroot:                     chrootDir,
	}

	return nil
}

func startCmdInSandbox(cmd *exec.Cmd) error {
	return cmd.Start()
}
