package sandbox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	BinaryPath string
	BinaryName string
	WorkDir    string
	Args       []string
}

type SandboxedProcess struct {
	cmd       *exec.Cmd
	stdoutBuf *bytes.Buffer
	cleanup   func() error
}

func NewSandboxedProcess(orig Config) (*SandboxedProcess, error) {
	cfg, cleanup, err := prepareSandbox(orig)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(cfg.BinaryPath, cfg.Args...)
	cmd.Dir = cfg.WorkDir
	cmd.Stdin = os.Stdin

	stdoutBuf := new(bytes.Buffer)
	stdoutWriter := io.MultiWriter(stdoutBuf, os.Stdout)
	stderrWriter := io.MultiWriter(stdoutBuf, os.Stderr)

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := applySandboxLimits(cmd); err != nil {
		cleanup()
		return nil, err
	}

	return &SandboxedProcess{
		cmd:       cmd,
		cleanup:   cleanup,
		stdoutBuf: stdoutBuf,
	}, nil
}

func (p *SandboxedProcess) Start() error {
	if runtime.GOOS == "windows" {
		return startCmdInSandbox(p.cmd)
	}
	return p.cmd.Start()
}

func (p *SandboxedProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *SandboxedProcess) Stop() error {
	proc := p.cmd.Process
	if proc == nil {
		return fmt.Errorf("没有可停止的进程")
	}

	var cleanupOnce sync.Once
	cleanup := func() error {
		var err error
		cleanupOnce.Do(func() {
			err = p.cleanup()
		})
		return err
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return cleanup()
	}

	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	var waitErr error
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("进程未能优雅退出，强制杀死")
		proc.Kill()
		waitErr = <-done
	case err := <-done:
		waitErr = err
	}

	if waitErr != nil && !strings.Contains(waitErr.Error(), "no child processes") {
		fmt.Printf("进程退出时出错: %v\n", waitErr)
	}

	return cleanup()
}

func (p *SandboxedProcess) StdoutBuffer() *bytes.Buffer {
	return p.stdoutBuf
}

func prepareSandbox(orig Config) (Config, func() error, error) {
	tmpRoot, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		return Config{}, nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	cleanup := func() error {
		fmt.Println("清理临时目录:", tmpRoot)
		if err := os.RemoveAll(tmpRoot); err != nil {
			return fmt.Errorf("删除 %s 失败: %w", tmpRoot, err)
		}
		fmt.Println("临时目录已清理")
		return nil
	}

	binName := filepath.Base(orig.BinaryPath)
	tmpBin := filepath.Join(tmpRoot, binName)
	if err := copyFile(orig.BinaryPath, tmpBin); err != nil {
		cleanup()
		return Config{}, nil, fmt.Errorf("拷贝二进制失败: %w", err)
	}

	if err := os.Chmod(tmpBin, 0755); err != nil {
		cleanup()
		return Config{}, nil, fmt.Errorf("设置执行权限失败: %w", err)
	}

	if err := copyDir(orig.WorkDir, tmpRoot); err != nil {
		cleanup()
		return Config{}, nil, fmt.Errorf("拷贝工作目录失败: %w", err)
	}

	return Config{
		BinaryPath: tmpBin,
		BinaryName: binName,
		WorkDir:    tmpRoot,
		Args:       orig.Args,
	}, cleanup, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}
