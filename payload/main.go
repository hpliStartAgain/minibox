package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
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
	case "ls":
		ents, err := os.ReadDir(arg(1, "/"))
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		fmt.Println(err, names)
	case "write":
		err := os.WriteFile(arg(1, "/x"), []byte("hi"), 0o644)
		fmt.Printf("写 %s -> %v\n", arg(1, "/x"), err)
	case "cat":
		b, err := os.ReadFile(arg(1, "/etc/passwd"))
		fmt.Printf("读 /proc/1/root -> %d 项 err=%v\n", len(b), err)
	case "proc1root":
		ents, err := os.ReadDir("/proc/1/root")
		fmt.Printf("读 /proc/1/root/ -> %d 项 err=%v\n", len(ents), err)
	case "chroot-escape":
		_ = os.MkdirAll("/tmp/esc", 0o755)
		if err := syscall.Chroot("/tmp/esc"); err != nil {
			fmt.Println("chroot ->", err)
			return
		}
		for range 8 {
			_ = syscall.Chdir("..")
		}
		err := syscall.Chroot(".")
		ents, _ := os.ReadDir("/")
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		fmt.Printf("chroot(.) -> %v; 现在看到的 / = %v\n", err, names)
	default:
		fmt.Println("unkonwn", args)
		os.Exit(64)
	}
}
