package box_test

import (
	"os"
	"testing"
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
