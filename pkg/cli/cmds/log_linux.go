//go:build linux && cgo
// +build linux,cgo

package cmds

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	systemd "github.com/coreos/go-systemd/daemon"
	"github.com/erikdubbelboer/gspt"
	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"github.com/wangxiaochuang/k3s/pkg/version"
	"golang.org/x/sys/unix"
)

func forkIfLoggingOrReaping() error {
	var stdout, stderr io.Writer = os.Stdout, os.Stderr
	enableLogRedirect := LogConfig.LogFile != "" && os.Getenv("_K3S_LOG_REEXEC_") == ""
	enableReaping := os.Getpid() == 1

	if enableLogRedirect {
		var l io.Writer = &lumberjack.Logger{
			Filename:   LogConfig.LogFile,
			MaxSize:    50,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}
		if LogConfig.AlsoLogToStderr {
			l = io.MultiWriter(l, os.Stderr)
		}
		stdout = l
		stderr = l
	}

	if enableLogRedirect || enableReaping {
		gspt.SetProcTitle(os.Args[0] + " init")

		pwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get working directory")
		}

		if enableReaping {
			unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0)
			go reapChildren()
		}

		args := append([]string{version.Program}, os.Args[1:]...)
		env := append(os.Environ(), "_K3S_LOG_REEXEC_=true", "NOTIFY_SOCKET=")
		cmd := &exec.Cmd{
			Path:   "/proc/self/exe",
			Dir:    pwd,
			Args:   args,
			Env:    env,
			Stdin:  os.Stdin,
			Stdout: stdout,
			Stderr: stderr,
			SysProcAttr: &syscall.SysProcAttr{
				Pdeathsig: unix.SIGTERM,
			},
		}
		if err := cmd.Start(); err != nil {
			return err
		}

		systemd.SdNotify(true, "READY=1\n")
		cmd.Wait()
		os.Exit(cmd.ProcessState.ExitCode())
	}

	return nil
}

func reapChildren() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGCHLD)
	for {
		select {
		case <-sigs:
		}
		for {
			var wstatus syscall.WaitStatus
			_, err := syscall.Wait4(-1, &wstatus, 0, nil)
			for err == syscall.EINTR {
				_, err = syscall.Wait4(-1, &wstatus, 0, nil)
			}
			if err == nil || err == syscall.ECHILD {
				break
			}
		}
	}
}
