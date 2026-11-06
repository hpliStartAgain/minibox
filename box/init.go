package box

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"
)

const initArg0 = "minibox-init"

func MaybeInit() {
	if len(os.Args) == 0 || os.Args[0] != initArg0 {
		return
	}

	runtime.LockOSThread()
	errF := os.NewFile(5, "initerr")
	syscall.CloseOnExec(5)
	if err := initStage(); err != nil {
		report(errF, err)
		os.Exit(125)
	}
	os.Exit(126)
}

func initStage() error {
	cfgF, syncF := os.NewFile(3, "cfg"), os.NewFile(4, "sync")
	if _, err := io.Copy(io.Discard, syncF); err != nil {
		return stage("sync", err)
	}
	var c Config
	if err := json.NewDecoder(cfgF).Decode(&c); err != nil {
		return stage("cfg", err)
	}

	cfgF.Close()
	syncF.Close()
	if len(c.Argv) == 0 {
		return stage("config", errors.New("empty argv"))
	}
	if c.Hostname != "" {
		if err := syscall.Sethostname([]byte(c.Hostname)); err != nil {
			return stage("sethostname", err)
		}
	}
	return stage("exec", syscall.Exec(c.Argv[0], c.Argv, c.Env))
}

type stageError struct {
	Op  string
	Err error
}

func (e *stageError) Error() string { return e.Op + ": " + e.Err.Error() }
func (e *stageError) Unwrap() error { return e.Err }

func stage(op string, err error) error {
	if err == nil {
		return nil
	}
	return &stageError{op, err}
}

type initReport struct {
	Op    string  `json:"op"`
	Errno uintptr `json:"errno"`
	Msg   string  `json:"msg"`
}

func report(f *os.File, err error) {
	r := initReport{Op: "init", Msg: err.Error()}
	var se *stageError
	if errors.As(err, &se) {
		r.Op = se.Op
	}
	var en syscall.Errno
	if errors.As(err, &en) {
		r.Errno = uintptr(en)
	}
	_ = json.NewEncoder(f).Encode(r)
}

func decodeInitError(b []byte) error {
	var r initReport
	if json.Unmarshal(b, &r) != nil {
		return errors.New("init: " + string(b))
	}
	if r.Errno != 0 {
		return &stageError{r.Op, syscall.Errno(r.Errno)}
	}
	return &stageError{r.Op, errors.New(r.Msg)}
}
