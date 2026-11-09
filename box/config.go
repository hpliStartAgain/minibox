package box

import "time"

type Config struct {
	Argv     []string
	Env      []string
	Timeout  time.Duration `json:"-"`
	Grace    time.Duration `json:"-"`
	Hostname string
	Faults   []string
	Rootfs   string
	ReadOnly bool
	Binds    map[string]string
	RWBinds  map[string]string
	Cwd      string
}

type Reason string

const (
	ReasonExited   Reason = "exited"
	ReasonSignaled Reason = "signaled"
	ReasonTimeout  Reason = "timeout"
)

type Result struct {
	Reason   Reason
	ExitCode int
	Signal   int
	Stdout   []byte
	Stderr   []byte
	Wall     time.Duration
}
