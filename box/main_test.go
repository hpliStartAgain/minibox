package box_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"example.com/minibox/box"
)

var (
	rootfs  string
	payload string
)

func TestMain(m *testing.M) {
	box.MaybeInit()
	dir, err := os.MkdirTemp("", "mbroot")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.Chmod(dir, 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "etc"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "etc", "hostname"), []byte("box\n"), 0o644)
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "bin", "payload"), "example.com/minibox/payload")
	build.Env = append(os.Environ(), "CGO_ENABLE=0")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build payload: %v\n%s", err, out)
		os.Exit(1)
	}
	rootfs, payload = dir, filepath.Join(dir, "bin", "payload")
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
