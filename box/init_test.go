package box_test

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"
)

func TestConfigSurvivesTheWire(t *testing.T) {
	arg := "héllo wörld\n\ttab \"quote\" 中文"
	cfg := sh(`printf %s "$1"`)
	cfg.Argv = append(cfg.Argv, "x", arg)
	res := run(t, cfg)
	if string(res.Stdout) != arg {
		t.Fatalf("stdout=%q, 期望 %q", res.Stdout, arg)
	}
	t.Logf("参数原样到达目标程序：%q", res.Stdout)
}

func TestInitFailureCarriesErrno(t *testing.T) {
	cfg := sh("true")
	cfg.Argv = []string{"/no/such/prog"}
	_, err := runErr(cfg)
	if !errors.Is(err, fs.ErrNotExist) || !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("err=%v", err)
	}
	t.Logf("exec 失败被原样带回父进程：%v", err)
	cfg.Argv = []string{"/etc/passwd"}
	_, err = runErr(cfg)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("err=%v", err)
	}
	t.Logf("没有执行权限：%v", err)
}

func TestNoLeakedDescriptors(t *testing.T) {
	res := run(t, sh(`for n in 3 4 5; do if [ -e /proc/$$/fd/$n ]; then echo "fd$n=$(readlink /proc/$$/fd/$n)"; fi; done; echo done`))
	if string(res.Stdout) != "done\n" {
		t.Fatalf("目标程序中发现了泄漏的 fd：%q", res.Stdout)
	}
	t.Logf("目标程序里 fd 3/4/5 都不存在：%q", res.Stdout)
}
