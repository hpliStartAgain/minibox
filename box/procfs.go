package box

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type ProcStat struct {
	PID, PPID, PGRP int
	Comm            string
	State           byte
	UTime, STime    uint64
	NumThreads      int
}

const clkTck = 100

func (p ProcStat) CPU() time.Duration {
	return time.Duration(p.UTime+p.STime) * time.Second / clkTck
}

func ReadStat(pid int) (ProcStat, error) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ProcStat{}, err
	}
	return ParseStat(string(b))
}

func ParseStat(s string) (ProcStat, error) {
	l, r := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if l < 0 || r < l {
		return ProcStat{}, fmt.Errorf("procfs: 格式不对：%q", s)
	}

	var p ProcStat
	var err error
	if p.PID, err = strconv.Atoi(strings.TrimSpace(s[:l])); err != nil {
		return ProcStat{}, fmt.Errorf("procfs: pid: %w", err)
	}

	p.Comm = s[l+1 : r]
	f := strings.Fields(s[r+1:])
	if len(f) < 18 {
		return ProcStat{}, fmt.Errorf("procfs: 字段太少 （%d）: %q", len(f), s)
	}
	atoi := func(i int) int { n, _ := strconv.Atoi(f[i]); return n }
	atou := func(i int) uint64 { n, _ := strconv.ParseUint(f[i], 10, 64); return n }
	p.State = f[0][0]
	p.PPID, p.PGRP = atoi(1), atoi(2)
	p.UTime, p.STime = atou(11), atou(12)
	p.NumThreads = atoi(17)
	return p, nil
}

func Alive(pid int) bool {
	p, err := ReadStat(pid)
	return err == nil && p.State != 'Z'
}

func FindByComm(name string) []int {
	ents, _ := os.ReadDir("/proc")
	var pids []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if p, err := ReadStat(pid); err == nil && p.Comm == name && p.State != 'Z' {
			pids = append(pids, pid)
		}
	}
	return pids
}

func StatusField(pid int, key string) string {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && k == key {
			return strings.TrimSpace(v)
		}
	}

	return ""
}
