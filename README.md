# minibox

`minibox` 是一个使用 Go 语言实现的轻量级 Linux 容器沙箱原型，用于系统性演练与理解现代容器运行时（如 runc、containerd）的核心隔离与生命周期管理机制。

## 核心特性

- **re-exec 架构与三管道同步协议**：
  - 突破 Go 语言多线程运行时与 POSIX `fork()` 的语义冲突，通过 `/proc/self/exe` 重新拉起干净的 Go 运行时。
  - 使用专用文件描述符握手：
    - `fd 3`（配置管道）：传输结构化 JSON 配置，规避命令行参数长度限制与 `/proc/<pid>/cmdline` 信息泄露；
    - `fd 4`（同步管道）：通过 EOF 信号原子放行子进程，确保宿主侧 Cgroup 入组与 UID 映射就绪后再执行目标代码；
    - `fd 5`（错误管道）：配合 `O_CLOEXEC` 实现零开销的执行成功检测与结构化系统错误码（`syscall.Errno`）回传。
- **Linux 命名空间（Namespace）隔离**：
  - **PID Namespace**：子进程成为空间内的 PID 1（PPID 显示为 0），杜绝 `setsid` 逃逸；
  - **Mount Namespace**：隔离文件系统挂载树，为后续换根（`pivot_root`）与私有 `procfs` 挂载提供基础；
  - **UTS Namespace**：独立主机名设置，不污染宿主机系统环境；
  - **IPC Namespace**：隔离 System V 消息队列与共享内存段；
  - **User Namespace**：演练非特权用户映射为局部 Root（Rootless 容器基础）。
- **进程树治理与信号控制**：
  - 支持会话组信号传递与 PID 1 级联销毁；
  - 覆盖超时宽限（Grace Period）与无处理函数信号（SIGTERM）丢弃行为。
- **procfs 自主解析**：
  - 内置高性能 `/proc/<pid>/stat` 与 `/proc/<pid>/status` 解析器，支持进程状态检测、CPU 耗时统计与异常名称（comm）容错。

## 目录结构

```text
minibox/
├── box/
│   ├── config.go       # 沙箱配置结构与运行参数
│   ├── fault.go        # 教学用故障注入开关
│   ├── init.go         # init 阶段逻辑与错误回传解码
│   ├── procfs.go       # procfs 进程信息读取与解析
│   ├── run.go          # 沙箱拉起、管道装配与退出码分类
│   └── *_test.go       # 单元测试与验证套件
├── payload/
│   └── main.go         # 沙箱内部运行的静态探测程序
└── go.mod
```

## 环境要求与运行测试

本项目依赖 Linux 内核专有特性（`CLONE_NEWPID`、`CLONE_NEWNS`、procfs 等），**必须在 Linux 环境下以 root 权限运行**。

### 运行全量测试

```bash
# 静态代码检查
go vet ./...

# 运行命名空间与核心测试
go test ./box -v -count=1 -run 'Namespace|PID1'

# 运行全量测试套件
go test ./box -v -count=1
```
