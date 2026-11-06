package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func countProcs() int {
	ents, _ := os.ReadDir("/proc")
	n := 0
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err == nil {
			n++
		}
	}
	return n
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"info"}
	}
	arg := func(i int, def string) string {
		if len(args) > i {
			return args[i]
		}
		return def
	}
	switch args[0] {
	case "info":
		h, _ := os.Hostname()
		cwd, _ := os.Getwd()
		fmt.Printf("pid=%d ppid=%d uid=%d gid=%d host=%s cwd=%s procs=%d\n",
			os.Getpid(), os.Getppid(), os.Getuid(), os.Getgid(), h, cwd, countProcs())
	case "sleep":
		s, _ := strconv.ParseFloat(arg(1, "1"), 64)
		time.Sleep(time.Duration(s * float64(time.Second)))
	case "exit":
		n, _ := strconv.Atoi(arg(1, "3"))
		os.Exit(n)
	case "escape":
		n, _ := strconv.Atoi(arg(1, "3"))
		for range n {
			c := exec.Command(os.Args[0], "sleep", "30")
			c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			_ = c.Start()
		}
		time.Sleep(30 * time.Second)
	default:
		fmt.Println("unkonwn", args)
		os.Exit(64)
	}
}
