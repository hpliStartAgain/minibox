package box_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/minibox/box"
)

func TestPIDNamespace(t *testing.T) {
	r := P(t, base(), "info")
	m := kv(string(r.Stdout))
	if m["pid"] != "1" || m["ppid"] != "0" {
		t.Fatalf("沙箱里 pid=%s ppid=%s，期望 1 和 0：%s", m["pid"], m["ppid"], txt(r))
	}
	t.Logf("沙箱里：%s", txt(r))
	if m["procs"] == "1" {
		t.Fatalf("procs=1? /proc 不应该是最新的")
	}
	t.Logf("沙箱里 getpid()==1，但 /proc 里仍然能数出 %s个进程（宿主的）", m["procs"])
}

func TestHostnameNamespace(t *testing.T) {
	before, _ := os.Hostname()
	r := P(t, base(), "info")
	after, _ := os.Hostname()
	if kv(string(r.Stdout))["host"] != "box" {
		t.Fatalf("沙箱里的主机名：%s", txt(r))
	}
	if before != after {
		t.Fatalf("主机名从 %q 变成了 %q", before, after)
	}
	t.Logf("沙箱里的主机名：box, 宿主 %q -> %q，没有变化", before, after)
}

func TestPID1IgnoreSignalWithoutHandler(t *testing.T) {
	c := base()
	c.Argv = []string{"/bin/sleep", "30"}
	c.Timeout, c.Grace = 200*time.Millisecond, time.Second
	r := run(t, c)
	if r.Reason != box.ReasonTimeout || r.Wall < 1000*time.Millisecond {
		t.Fatalf("reason=%s wall=%v, 期望 TERM 被忽略、等满宽限期才被 KILL", r.Reason, r.Wall)
	}
	t.Logf("PID 1 的 sleep 忽略了 SIGTERM：wall=%v ≈ 200ms 超时 + 1s 宽限（docker stop 要等 10 秒就是这个原因）", r.Wall.Round(10*time.Millisecond))
}

func uniqueCopy(t *testing.T) (path, comm string) {
	t.Helper()
	b, err := os.ReadFile(payload)
	if err != nil {
		t.Fatal(err)
	}
	comm = fmt.Sprintf("pb%d", time.Now().UnixNano()%1e12)
	path = filepath.Join(t.TempDir(), comm)
	if err := os.WriteFile(path, b, 0o755); err != nil {
		t.Fatal(err)
	}

	return path, comm
}

func TestPIDNamespaceCleansEscapedProcesses(t *testing.T) {
	path, comm := uniqueCopy(t)
	cmd := exec.Command(path, "escape", "3")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	_ = cmd.Start()
	time.Sleep(200 * time.Millisecond)
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Wait()
	time.Sleep(100 * time.Millisecond)
	leaked := box.FindByComm(comm)
	if len(leaked) != 3 {
		t.Fatalf("对照组应该泄漏 3 个, 实际 %v", leaked)
	}
	t.Logf("没有命名空间：杀掉进程组后, 仍有 %d 个逃逸的 payload 在跑 %v", len(leaked), leaked)
	for _, pid := range leaked {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	time.Sleep(100 * time.Millisecond)

	c := base()
	c.Timeout = 400 * time.Millisecond
	c.Argv = []string{path, "escape", "3"}
	r := run(t, c)
	time.Sleep(100 * time.Millisecond)
	if r.Reason != box.ReasonTimeout || len(box.FindByComm(comm)) != 0 {
		t.Fatalf("reason=%s 残留=%v", r.Reason, box.FindByComm(comm))
	}
	t.Logf("有命名空间：超时后宿主上残留的 %s 进程 = %d", comm, len(box.FindByComm(comm)))
}

func TestUserNamespace(t *testing.T) {
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skip("没有 unshare")
	}

	script := `echo "host-uid=$(id -u)"; unshare -U -r -m sh -c 'echo "ns-uid=$(id -u)"; cat /proc/self/uid_map; mkdir -p /tmp/uns-$$ && mount -t tmpfs t /tmp/uns-$$ && echo mount=ok'`
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
	out, err := cmd.CombinedOutput()
	s := string(out)
	if err != nil || !strings.Contains(s, "host-uid=65534") || !strings.Contains(s, "ns-uid=0") || !strings.Contains(s, "mount=ok") {
		t.Fatalf("此内核/配置不允许非特权用户命名空间: err=%v out=%s", err, s)
	}
	t.Logf("非特权用户创建用户命名空间: \n%s", s)
}
