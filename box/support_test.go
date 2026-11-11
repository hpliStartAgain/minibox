package box_test

import (
	"strings"
	"testing"
	"time"

	"example.com/minibox/box"
)

func base() box.Config {
	return box.Config{
		Env: []string{
			"PATH=/bin:/usr/bin",
		},
		Timeout:  10 * time.Second,
		Hostname: "box"}
}

func P(t *testing.T, cfg box.Config, args ...string) box.Result {
	t.Helper()
	exe := payload
	if cfg.Rootfs != "" {
		exe = "/bin/payload"
	}
	cfg.Argv = append([]string{exe}, args...)
	return run(t, cfg)
}

func kv(s string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Fields(s) {
		if k, v, ok := strings.Cut(f, "="); ok {
			m[k] = v
		}
	}
	return m
}

func txt(r box.Result) string {
	s := strings.TrimSpace(string(r.Stdout) + string(r.Stderr))
	return strings.ReplaceAll(s, "\n", " ⏎ ")
}
