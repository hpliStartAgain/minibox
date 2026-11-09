package box

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type pipe struct {
	r, w *os.File
}

func newPipe() (pipe, error) {
	r, w, err := os.Pipe()
	return pipe{r, w}, err
}

func (p pipe) close() {
	if p.r != nil {
		p.r.Close()
	}
	if p.w != nil {
		p.w.Close()
	}
}

func Run(ctx context.Context, cfg Config) (Result, error) {
	var res Result
	if len(cfg.Argv) == 0 {
		return res, errors.New("empty argv")
	}
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	grace := cfg.Grace
	if grace == 0 {
		grace = 500 * time.Millisecond
	}

	cfgP, err := newPipe()
	if err != nil {
		return res, err
	}
	defer cfgP.close()
	syncP, err := newPipe()
	if err != nil {
		return res, err
	}
	defer syncP.close()
	errP, err := newPipe()
	if err != nil {
		return res, err
	}
	defer errP.close()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "/proc/self/exe")
	cmd.Args = []string{initArg0}
	cmd.Env = []string{}
	cmd.ExtraFiles = []*os.File{cfgP.r, syncP.r, errP.w}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	flags := uintptr(syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC)
	if !cfg.fault("no-pidns") {
		flags |= syscall.CLONE_NEWPID
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Cloneflags: flags, Pdeathsig: syscall.SIGKILL}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	cmd.WaitDelay = grace

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return res, fmt.Errorf("start init: %w", err)
	}
	cfgP.r.Close()
	syncP.r.Close()
	errP.w.Close()

	if err := json.NewEncoder(cfgP.w).Encode(cfg); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return res, fmt.Errorf("send config: %w", err)
	}

	cfgP.w.Close()
	syncP.w.Close()
	initErr, _ := io.ReadAll(errP.r)

	_ = cmd.Wait()
	res.Wall = time.Since(start)
	res.Stdout, res.Stderr = stdout.Bytes(), stderr.Bytes()
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

	if len(initErr) > 0 {
		return res, decodeInitError(initErr)
	}
	ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
	switch {
	case ctx.Err() != nil:
		res.Reason = ReasonTimeout
	case ws.Signaled():
		res.Reason, res.Signal = ReasonSignaled, int(ws.Signal())
	default:
		res.Reason, res.ExitCode = ReasonExited, ws.ExitStatus()
	}
	return res, nil
}
