package box_test

import (
	"testing"

	"example.com/minibox/box"
)

func TestParseStatWeirdComm(t *testing.T) {
	line := "42 (a b) c) S 1 42 42 0 -1 4194560 100 0 0 0 7 3 0 0 20 0 2 0 123456 1000 10"
	p, err := box.ParseStat(line)
	if err != nil {
		t.Fatal(err)
	}

	if p.PID != 42 || p.Comm != "a b) c" || p.State != 'S' || p.PPID != 1 || p.PGRP != 42 || p.UTime != 7 || p.STime != 3 || p.NumThreads != 2 {
		t.Fatalf("解析错误：%+v", p)
	}
	t.Logf("解析结果：%+v  CPU（）=%v", p, p.CPU())
	if _, err := box.ParseStat("garbage"); err == nil {
		t.Fatal("坏格式应该解析失败")
	}
}
