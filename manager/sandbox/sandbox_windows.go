//go:build windows

package sandbox

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

func applySandboxLimits(_ *exec.Cmd) error {
	return nil
}

func startCmdInSandbox(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		cmd.Process.Kill()
		return err
	}

	info := windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
		LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
	}

	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectBasicLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		cmd.Process.Kill()
		return err
	}

	return windows.AssignProcessToJobObject(job, windows.Handle(cmd.Process.Pid))
}
