//go:build darwin

package sandbox

import "os/exec"

func applySandboxLimits(cmd *exec.Cmd) error {
	profile := "(version 1) (allow default)"
	cmd.Args = append([]string{"sandbox-exec", "-p", profile, "--", cmd.Path}, cmd.Args...)
	cmd.Path = "sandbox-exec"
	return nil
}

func startCmdInSandbox(cmd *exec.Cmd) error {
	return cmd.Start()
}
