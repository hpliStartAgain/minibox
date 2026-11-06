package box_test

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/minibox/box"
)

func pidFromStdout(t *testing.T, out []byte) int {
	t.Helper()
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("stdout=%q: %v", out, err)
	}
	return pid
}

func TestTimeoutKillsWholeGroup(t *testing.T) {
	cfg := sh("sleep 30 & echo $!; wait")
	cfg.Timeout, cfg.Grace = 200*time.Millisecond, 300*time.Millisecond
	res := run(t, cfg)
	pid := pidFromStdout(t, res.Stdout)
	if res.Reason != box.ReasonTimeout {
		t.Fatalf("reason=%s", res.Reason)
	}
	if !(waitDead(pid, time.Second)) {
		t.Fatalf("孙进程 %d 还活着: 只杀了组长，没杀整组", pid)
	}
	t.Logf("超时后孙进程 %d 已死 （wall=%v）", pid, res.Wall.Round(10*time.Millisecond))
}

func TestBackgroudJobDoesNotOutliveRun(t *testing.T) {
	res := run(t, sh("sleep 30 & echo $!"))
	pid := pidFromStdout(t, res.Stdout)
	if res.Reason != box.ReasonExited || res.ExitCode != 0 {
		t.Fatalf("reason=%s exit=%d", res.Reason, res.ExitCode)
	}
	if !waitDead(pid, time.Second) {
		t.Fatalf("孤儿 %d 在 Run 饭后后仍然活着", pid)
	}
	t.Logf("孤儿 %d 已被清理；Run 耗时 %v （孤儿握着管道，被 WaitDelay 拖住了约 500ms）", pid, res.Wall.Round(10*time.Millisecond))
}

func TestWithoutGroupKillItLeaks(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "sleep 30 >/dev/null 2>&1 & echo $!")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	pid := pidFromStdout(t, out)
	defer syscall.Kill(pid, syscall.SIGKILL)
	st, ppid, ok := procStat(pid)
	if !ok || st == 'Z' {
		t.Fatalf("孙进程一个还活着：ok=%v state=%c", ok, st)
	}
	if ppid == cmd.Process.Pid {
		t.Fatalf("孙进程的父进程还是已退出的 sh（%d）? ", ppid)
	}
	t.Logf("sh(%d) 已退出，孙进程 %d 仍在运行，状态=%c，新的父进程=%d（被收养）", cmd.Process.Pid, pid, st, ppid)
}

func TestZombieUntilWaited(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	pid := cmd.Process.Pid
	var st byte
	for end := time.Now().Add(2 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		if s, _, ok := procStat(pid); ok && s == 'Z' {
			st = s
			break
		}
	}
	if st != 'Z' {
		t.Fatalf("子进程已退出但没人 Wait, 应该是僵尸 Z, 实际 %c", st)
	}
	t.Logf("父进程还没 Wait：/proc/%d 的状态 = %c （僵尸）", pid, st)
	_ = cmd.Wait()
	if _, _, ok := procStat(pid); ok {
		t.Fatalf("Wait 之后 /proc 里不应该还有这个进程")
	}
	t.Logf("Wait 之后：/proc/%d 消失了", pid)
}
