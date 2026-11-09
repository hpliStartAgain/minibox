package box

import (
	"os"
	"path/filepath"
	"syscall"
)

func setupRoot(c Config) error {
	root := c.Rootfs
	// pivot_root 要求新根是一个挂载点：把它 bind 到自己身上
	if err := syscall.Mount(root, root, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return stage("bind rootfs", err)
	}

	// 必须在 pivot 之前挂好 /proc（它需要属于新的 PID 命名空间）、/dev、/tmp，并建好存放旧根的 .pivot。
	for _, d := range []string{"proc", "dev", "tmp", ".pivot"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return stage("mkdir "+d, err)
		}
	}

	if !c.fault("no-proc") {
		if err := syscall.Mount("proc", filepath.Join(root, "proc"), "proc", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, ""); err != nil {
			return stage("mount proc", err)
		}
	}
	if err := syscall.Mount("tmpfs", filepath.Join(root, "dev"), "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC, "mode=755, size=64k"); err != nil {
		return stage("mount dev", err)
	}

	// 只放 4 个最基本的设备：从宿主 /dev 里 bind 过来。
	for _, n := range []string{"null", "zero", "urandom", "random"} {
		dst := filepath.Join(root, "dev", n)
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o666)
		if err != nil {
			return stage("create /dev/"+n, err)
		}
		f.Close()
		if err := syscall.Mount("/dev/"+n, dst, "", syscall.MS_BIND, ""); err != nil {
			return stage("bind /dev/"+n, err)
		}
	}
	if err := syscall.Mount("tmpfs", filepath.Join(root, "tmp"), "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV, "mode=1777, size=16m"); err != nil {
		return stage("mount tmp", err)
	}
	for src, dst := range c.Binds {
		target := filepath.Join(root, dst)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return stage("mkdir "+dst, err)
		}
		if err := syscall.Mount(src, target, "", syscall.MS_BIND, ""); err != nil {
			return stage("bind "+src, err)
		}
		// 绑定挂载变成只读需要二次 remount 才生效；第一次的 RDONLY 标志会被内核忽略
		if err := syscall.Mount("", target, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			return stage("bind "+src, err)
		}
	}
	for src, dst := range c.RWBinds {
		target := filepath.Join(root, dst)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return stage("mkdir "+dst, err)
		}
		if err := syscall.Mount(src, target, "", syscall.MS_BIND, ""); err != nil {
			return stage("bind "+src, err)
		}
		// 读写绑定也有 remount 一次，加上 nosuid 和 nodev； 栏目的 setuid 文件和设备节点不能生效
		if err := syscall.Mount("", target, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_NOSUID|syscall.MS_NODEV, ""); err != nil {
			return stage("bind "+src, err)
		}
	}
	if c.fault("chroot") {
		// 故障版本：只改根目录指针，不换挂载树
		if err := syscall.Chroot(root); err != nil {
			return stage("chroot", err)
		}
		if err := syscall.Chdir("/"); err != nil {
			return stage("chdir", err)
		}
	} else {
		if err := syscall.PivotRoot(root, filepath.Join(root, ".pivot")); err != nil {
			return stage("pivot_root", err)
		}
		if err := syscall.Chdir("/"); err != nil {
			return stage("chdir", err)
		}
		// 旧根必须卸载并删除，否则沙箱里还能通过 /.pivot 摸到宿主文件系统
		if !c.fault("keep-old-root") {
			if err := syscall.Unmount("/.pivot", syscall.MNT_DETACH); err != nil {
				return stage("unmount old root", err)
			}
			_ = os.Remove("/.pivot")
		}
	}
	if c.ReadOnly {
		if err := syscall.Mount("", "/", "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			return stage("remount / ro", err)
		}
	}
	return nil
}
