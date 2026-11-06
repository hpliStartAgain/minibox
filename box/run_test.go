package box_test

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"testing"
	"time"

	"example.com/minibox/box"
)

func sh(script string) box.Config {
	return box.Config{
		Argv:   []string{"/bin/sh", "-c", script},
		Env:    []string{"PATH=/usr/bin:/bin"},
		Faults: []string{"no-pids"},
	}
}

func run(t *testing.T, cfg box.Config) box.Result {
	t.Helper()
	res, err := box.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

func runErr(cfg box.Config) (box.Result, error) {
	return box.Run(context.Background(), cfg)
}

func TestExitCode(t *testing.T) {
	res := run(t, sh("echo hello; echo oops >&2; exit 3"))
	if res.Reason != box.ReasonExited || res.ExitCode != 3 {
		t.Fatalf("reason=%s exit=%d", res.Reason, res.ExitCode)
	}

	if string(res.Stdout) != "hello\n" || string(res.Stderr) != "oops\n" {
		t.Fatalf("stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}

	t.Logf("reason=%s exit=%d stdout=%q stderr=%q", res.Reason, res.ExitCode, res.Stdout, res.Stderr)
}

func TestKilledBySignal(t *testing.T) {
	res := run(t, sh("kill -9 $$"))
	if res.Reason != box.ReasonSignaled || res.Signal != 9 {
		t.Fatalf("reason=%s signal=%d", res.Reason, res.Signal)
	}
	t.Logf("kill -9 -> reason=%s signal=%d exit=%d", res.Reason, res.Signal, res.ExitCode)
	res = run(t, sh("exit 255"))
	if res.Reason != box.ReasonExited || res.ExitCode != 255 {
		t.Fatalf("reason=%s exit=%d", res.Reason, res.ExitCode)
	}
	t.Logf("exit 255 -> reason=%s exit=%d", res.Reason, res.ExitCode)
}

func TestTimeoutTermThenKill(t *testing.T) {
	cfg := sh("trap '' TERM; exec sleep 5")
	cfg.Timeout, cfg.Grace = 200*time.Millisecond, 300*time.Millisecond
	res := run(t, cfg)
	if res.Reason != box.ReasonTimeout {
		t.Fatalf("reason=%s", res.Reason)
	}
	if res.Wall < 450*time.Millisecond || res.Wall > 1500*time.Millisecond {
		t.Fatalf("wall=%v, 应约等于 Timeout+Grace=500ms", res.Wall)
	}
	t.Logf("忽略TERM: reason=%s wall=%v (≈ 200ms 超时 + 300ms 宽限)", res.Reason, res.Wall.Round(10*time.Millisecond))

	cfg = sh("exec sleep 5")
	cfg.Timeout, cfg.Grace = 200*time.Millisecond, 300*time.Millisecond
	res = run(t, cfg)
	if res.Reason != box.ReasonTimeout || res.Wall > 450*time.Millisecond {
		t.Fatalf("reason=%s wall=%v", res.Reason, res.Wall)
	}
	t.Logf("响应TERM: reason=%s wall=%v", res.Reason, res.Wall.Round(10*time.Millisecond))
}

func TestGrandchildHoldsPipe(t *testing.T) {
	cfg := sh("sleep 2 & wait")
	cfg.Timeout, cfg.Grace = 200*time.Millisecond, 300*time.Millisecond
	res := run(t, cfg)
	if res.Reason != box.ReasonTimeout || res.Wall > time.Second {
		t.Fatalf("reason=%s wall=%v", res.Reason, res.Wall)
	}
	t.Logf("孙进程持有管道: Run 在 %v 返回", res.Wall.Round(10*time.Millisecond))
}

func TestEnvNotInherited(t *testing.T) {
	t.Setenv("SECRET_TOKEN", "hunter2")
	res := run(t, sh(`echo "token=${SECRET_TOKEN:-unset}"`))
	if string(res.Stdout) != "token=unset\n" {
		t.Fatalf("stdout=%q, 宿主环境变量泄漏进了子进程", res.Stdout)
	}
	out, _ := exec.Command("/bin/sh", "-c", `echo "token=${SECRET_TOKEN:-unset}"`).Output()
	t.Logf("box.Run → %q；对照：直接 exec.Command 默认继承 → %q", res.Stdout, out)
}

func TestStartFailures(t *testing.T) {
	ctx := context.Background()
	_, err := box.Run(ctx, box.Config{Argv: []string{"/no/such/prog"}})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("不存在的程序: %v", err)
	}
	t.Logf("不存在: %v", err)
	_, err = box.Run(ctx, box.Config{Argv: []string{"/etc/passwd"}})
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("不可执行的文件: %v", err)
	}
	t.Logf("不可执行: %v", err)
	if _, err = box.Run(ctx, box.Config{}); err == nil {
		t.Fatalf("空argv应该失败")
	}
	t.Logf("空argv: %v", err)
}
