# lane

高效、低成本地拉起独立的本地工作区，供多个 Agent 并行开发。

一个工作区包括 Git worktree、独立的数据目录、端口分配，以及一组项目声明的进程。lane 不理解某一种数据库，也不绑定某一种 Agent harness。

## 两种明确的运行方式

| 模式 | 适用场景 | 边界 |
| --- | --- | --- |
| `native`（默认） | 项目已经支持参数、配置文件或环境变量指定端口和数据路径 | 运行宿主原生进程；资源分区，不提供透明网络隔离 |
| `container` | 多个工作区需要保留相同的硬编码监听端口 | 每工作区一个可复用 Linux 容器；服务、测试、迁移和 hooks 进入同一容器 |

容器模式复用本机已有的 Docker/Podman Linux 引擎与预构建镜像。macOS/Windows 可以使用引擎管理的共享 Linux VM，**不是每个工作区再创建一个 VM，也不是每次运行都重新构建镜像**。引擎和镜像缺失会报错，不会静默退回宿主执行。

这些环境用于协作开发，不是运行恶意代码的安全沙箱。容器可写工作树与共享 Git 元数据；共享凭据、绝对路径、宿主工具和平台专用程序不会自动虚拟化。详见 [运行契约](docs/runtime.md)。

## 安装与原生模式

```sh
go install github.com/Mrjwj34/lane/cmd/lane@latest
# 在项目中：
lane init
```

编辑并提交项目的 `lane.yaml`。新工作区从指定 Git 基线检出，并使用**自己的**配置，不读取主工作树里未提交的运行配置。

```yaml
version: 1
base: main
runtime:
  backend: native
ports: [web]
env:
  PORT: ${LANE_PORT_WEB}
processes:
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
```

```sh
lane new feature-a --up
lane new feature-b --up
lane ls --json
lane plan feature-a
```

没有 `lane.yaml` 时，`new/ls/attach/done` 仍可用于纯 Git worktree 生命周期，无需容器或进程编排器。

## 不改硬编码端口的隔离模式

先在 **lane 源码目录** 构建一次基础镜像。它包含 bash、Git、Python、socat 和固定版本的 process-compose：

```sh
docker build -t lane-runtime:local -f runtime/Dockerfile .
# Go 项目可以使用包含 Go 工具链的基础镜像：
docker build --build-arg BASE_IMAGE=golang:1.25-bookworm \
  -t lane-go:local -f runtime/Dockerfile .
```

也可以为项目制作自己的镜像。使用稳定镜像引用或 digest；依赖与工具链在镜像构建时准备，不在每个 workspace 启动时重新安装。

项目配置：

```yaml
version: 1
base: main
runtime:
  backend: container
  engine: docker                  # 也接受 podman；必须是可访问工作树的本地 Linux 引擎
  image: lane-runtime:local        # 必须已存在；lane 不自动 pull/build
  memory: 2g                      # 可选上限，不是每工作区预留内存
  cpus: 2
ports: [web]
listen:
  web: 8080                       # 应用原来的 TCP 监听端口，30000 等也可以
hooks:
  setup: []                       # 在容器中运行，失败后会重试，必须可重复执行
  teardown: []
processes:
  web:
    command: python3 -m http.server 8080 --bind 127.0.0.1
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: 8080
        path: /
```

两个工作区都可以保留内部 `127.0.0.1:8080`。宿主访问各自独立的发布端口；内部回环转发支持只绑定 loopback 的应用，不拦截 `bind/connect` 系统调用。

```sh
lane new feature-a --up
lane new feature-b --up
lane open web                     # 在对应工作区目录运行
lane run -- python3 -c 'import urllib.request; print(urllib.request.urlopen("http://127.0.0.1:8080").status)'
```

`lane run` 的命令在当前工作区的同一执行环境中运行。`argv` 不被二次解析；需要 shell 时显式使用 `lane run -- sh -c '...'`。容器命令取消时停止该工作区容器，确保不会只终止 `docker exec` 客户端而留下目标命令。

## 环境变量

`LANE_WORKSPACE`、`LANE_ROOT`、`LANE_DATA_DIR` 和 `LANE_PORT_<NAME>` 对应**当前执行环境**。容器内的数据目录为 `/workspace/.lane/data`，`LANE_PORT_WEB` 为原始监听端口；原生模式对应宿主路径和分配端口。

`LANE_HOST_PORT_<NAME>` 始终是宿主发布端口，适合浏览器 URL 或前端构建参数。`attach` 输出宿主可用的环境变量；`ports --json`、`status --json`、`plan` 显示资源契约。发布端口当前仅支持 IPv4 回环上的 TCP；UDP/IPv6 不作为宿主发布契约。内部网络命名空间不受这个发布协议限制。

## 生命周期与安全

`new` 的初始化进度写入注册表。setup 失败后保留工作树，重试同名 `new` 或 `up` 会重试未完成的 setup。`down` 停止进程/容器但保留数据；容器下次启动复用同一实例。`reset` 确认运行环境停止后才重建数据与执行 setup。

`done` 删除 lane 创建的工作树前，必须确认工作树干净，且提交已在基线或 upstream 中保存。`--force` 仅跳过内容保留检查，**不能跳过主工作树保护、身份检查或进程停止检查**。

`adopt` 接管外部工具创建的 linked worktree。之后 `done` 只停止并注销运行环境，保留其工作树、数据和分支，即使指定 `--force`。主工作树不允许 adopt。旧版注册记录不会被自动授予删除权限；可在对应目录显式 `adopt` 完成安全迁移。

跨进程端口分配与登记在一个事务中完成。长操作使用工作区级锁，不持有全局注册表锁。注册表租约不是操作系统 socket 保留；外部进程抢占端口时启动失败，不伪装成功。

依赖复制使用 macOS clonefile / Linux reflink，不支持时普通复制，**不会以硬链接共享可写依赖**。外部符号链接不作为独立复制的保证范围。

## 清理与诊断

GC 只在显式调用时执行，默认策略关闭：

```yaml
gc:
  idle_stop_hours: 0
  remove_after_days: 0
  max_workspaces: 0
```

```sh
lane gc --dry-run --json
lane gc
lane doctor --json
lane doctor --fix                 # 仅准备原生依赖，不执行破坏性 GC
```

GC 跳过正在执行命令的工作区，重新检查身份和提交保留条件，永不使用强制删除。空闲时间基于 lane 操作，不是 CPU 使用率、浏览器流量或编辑器活动；启用空闲停机前应理解这个边界。状态不确定时保留数据并给出 warning，而不是猜测“已经停止”。

## 命令

`init`、`skill install`、`hook install`、`new`、`adopt`、`attach`、`ls`、`status`、`ports`、`plan`、`up`、`down`、`run`、`logs`、`reset`、`done`、`gc`、`doctor`、`open`。

## 开发与验证

```sh
go test ./cmd/... ./internal/...
go test -race ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go build ./cmd/lane
```

CI 在 Linux/macOS/Windows 运行核心与原生服务测试，并在 Linux 运行真实容器端到端测试：同端口并行、宿主回环不串线、相同 runtime 中的测试命令、Git worktree、独立数据、实例复用、取消与清理。

macOS/Windows 的容器引擎行为还需在实际 Docker Desktop/Podman Machine 环境验收；交叉编译不是端到端验证。测试范围与边界见 [架构记录](docs/architecture.md)。
