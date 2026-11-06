package box_test

import (
	"time"

	"example.com/minibox/box"
)

func procStat(pid int) (state byte, ppid int, ok bool) {
	p, err := box.ReadStat(pid)
	if err != nil {
		return 0, 0, false
	}
	return p.State, p.PPID, true
}

func alive(pid int) bool {
	return box.Alive(pid)
}

func waitDead(pid int, d time.Duration) bool {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if !alive(pid) {
			return true
		}
	}
	return !alive(pid)
}
