package route

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"sparkle-service/listen"
)

var (
	unixServer *http.Server
	pipeServer *http.Server
	secret     = "key"
)

func start() error {
	// if runtime.GOOS == "windows" {
	// if err = startServer("\\\\.\\pipe\\party-service", startPipe); err != nil {
	// 	return err
	// }
	// if err := startServer("./party-service.sock", StartUnix); err != nil {
	// 	return err
	// }
	if runtime.GOOS == "windows" {
		if err := startServer("127.0.0.1:10001", StartHTTP); err != nil {
			return err
		}
	} else {
		if err := startServer("127.0.0.1:10010", StartHTTP); err != nil {
			return err
		}
	}
	// } else {
	// 	if err := startServer("/tmp/sparkle-service.sock", StartUnix); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

func startServer(addr string, startFunc func(string) error) error {
	if unixServer != nil {
		_ = unixServer.Close()
		unixServer = nil
	}

	if pipeServer != nil {
		_ = pipeServer.Close()
		pipeServer = nil
	}

	if len(addr) > 0 {
		dir := filepath.Dir(addr)
		if err := ensureDirExists(dir); err != nil {
			return err
		}

		if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unlink error: %w", err)
		}

		if err := startFunc(addr); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("directory creation error: %w", err)
		}
	}
	return nil
}

func StartHTTP(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http listen error: %w", err)
	}
	log.Printf("http listening at: %s", addr)
	server := &http.Server{
		Handler: router(),
	}
	return server.Serve(l)
}

func StartUnix(addr string) error {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("unix listen error: %w", err)
	}
	_ = os.Chmod(addr, 0o666)
	log.Printf("unix listening at: %s", l.Addr().String())

	server := &http.Server{
		Handler: router(),
	}
	unixServer = server
	return server.Serve(l)
}

func StartPipe(addr string) error {
	if !strings.HasPrefix(addr, "\\\\.\\pipe\\") {
		return fmt.Errorf("windows namedpipe must start with \"\\\\.\\pipe\\\"")
	}

	l, err := listen.ListenNamedPipe(addr)
	if err != nil {
		return fmt.Errorf("pipe listen error: %w", err)
	}
	log.Printf("pipe listening at: %s", l.Addr().String())

	server := &http.Server{
		Handler: router(),
	}
	pipeServer = server
	return server.Serve(l)
}
