# lane

高效、低成本地拉起独立的本地工作区，供多个 Agent 并行开发。

一个工作区包括 Git worktree、独立的数据目录、端口分配，以及一组项目声明的进程。lane 不理解某一种数据库，也不绑定某一种 Agent harness。

---

## 核心特性

- **去中心化无守护进程（Daemonless）**：不驻留后台守护进程，依赖局部文件锁与状态持久化保证跨进程原子性。
- **两种明确的执行后端**：
  - **Native（默认）**：直接运行宿主原生进程，极低资源开销；通过环境变量与动态端口分配隔离不同工作区。
  - **Container**：复用本机已有的 Docker/Podman Linux 容器；内部通过 Loopback 端口转发支持硬编码监听端口（如 8080/5432），消除端口冲突。
- **严格的数据安全防误删（Anti-footgun）**：
  - 严禁删除或接管 Git 主工作树（Primary Checkout）；
  - 严禁删除含有未推送提交或脏代码的工作区（即便指定 `--force` 也会受到多重身份与状态阻断）；
  - 外部接管的 `adopt` 工作区在 `done` 时仅注销运行态，永远保留代码与分支；
  - 自动 GC 永远以保守策略运行，绝不静默强制删除。
- **开箱即用的 Agent 支持**：
  - 内置 `SKILL.md`，支持一键安装至 Claude Code、Cursor 和各类 Coding Agent；
  - 提供 Cursor Worktree 和 Claude Code 的生命周期集成 Hook。

---

## 安装

### 方式 1: GitHub Releases 二进制下载（推荐，免 Go 环境）

直接从 [GitHub Releases](https://github.com/Mrjwj34/lane/releases) 下载对应操作系统的预编译二进制归档，解压后将 `lane` 放置在系统 `PATH` 路径即可：
- **Linux** (`x86_64` / `arm64`): `lane_<version>_linux_<arch>.tar.gz`
- **macOS** (`Intel` / `Apple Silicon`): `lane_<version>_darwin_<arch>.tar.gz`
- **Windows** (`x86_64` / `arm64`): `lane_<version>_windows_<arch>.zip`

### 方式 2: Go Install（需 Go 1.25+）

```sh
go install github.com/Mrjwj34/lane/cmd/lane@latest
```

安装完成后可通过 `lane version` 验证安装成功及版本信息。

### 方式 3: 从源码构建

```sh
git clone https://github.com/Mrjwj34/lane.git
cd lane
go build -o bin/lane ./cmd/lane
# 将 ./bin 添加到 PATH 或复制到系统目录
```

---

## 快速上手与 Agent 工作流

### 1. 项目接入与 Agent Skill 安装

在项目根目录下执行：

```sh
# 生成默认 lane.yaml 模板并自动将 SKILL.md 分发至各 Agent 目录
lane init

# 如需为 Cursor 或 Claude Code 安装 worktree 联动 hook：
lane hook install all
```

执行后将自动安装：
- `.agents/skills/lane/SKILL.md`
- `.claude/skills/lane/SKILL.md`
- `.cursor/skills/lane/SKILL.md`

### 2. 编写 `lane.yaml`

编辑并提交项目根目录下的 `lane.yaml`。每个新工作区在检出时使用其**分支自身**的配置，不读取主工作树中未提交的草稿。

#### 原生模式示例（支持动态端口配置的项目）

```yaml
version: 1
base: main
ports: [web, pg]
env:
  PORT: ${LANE_PORT_WEB}
  DATABASE_URL: postgres://127.0.0.1:${LANE_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/pg"
    - test -d "$LANE_DATA_DIR/pg/base" || initdb -D "$LANE_DATA_DIR/pg" --no-locale --encoding=UTF8
processes:
  pg:
    command: postgres -D "$LANE_DATA_DIR/pg" -p "$LANE_PORT_PG" -k "$LANE_DATA_DIR"
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$LANE_PORT_PG"
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get: {host: 127.0.0.1, port: "${LANE_PORT_WEB}", path: /}
```

#### 容器模式示例（硬编码监听端口，如固定 8080）

容器模式复用本地 Docker/Podman 引擎与预构建镜像，每个工作区启动一个轻量 Linux 容器实例。
先在 **lane 源码目录** 或项目自身构建一次镜像：

```sh
docker build -t lane-runtime:local -f runtime/Dockerfile .
```

在项目中配置：

```yaml
version: 1
base: main
runtime:
  backend: container
  engine: docker                   # 或 podman
  image: lane-runtime:local         # 镜像必须已存在，lane 绝不隐式 pull/build
ports: [web]
listen:
  web: 8080                        # 应用原始硬编码监听端口
processes:
  web:
    command: python3 -m http.server 8080 --bind 127.0.0.1
    readiness_probe:
      http_get: {host: 127.0.0.1, port: 8080, path: /}
```

### 3. 创建与并行开发

```sh
# 在全新独立工作区检出分支并启动进程
lane new feature-a --up

# 并行创建第二个工作区（端口自动隔离，互不冲突）
lane new feature-b --up

# 查看所有工作区状态（支持 --json 给 Agent 消费）
lane ls --json
lane status feature-a
```

### 4. 工作区环境内执行命令

`lane run` 将命令直接运行在对应工作区的同一执行上下文中（注入对应环境变量）：

```sh
cd /path/to/feature-a
# 运行迁移或测试（argv 严格保持，不被二次转义）
lane run -- pytest
# 浏览器打开已映射的发布端口
lane open web
# 查看某进程实时日志
lane logs web
```

### 5. 重置、停止与清理

```sh
# 清空当前工作区的私有数据目录并重新执行 setup hook
lane reset

# 停止工作区进程（保留数据与容器实例，下次 up 快速恢复）
lane down

# 开发完成，提交或合并后安全销毁（严格检查干净度与已保存状态）
lane done

# 定期保守清理废弃工作区（跳过运行中和有未保存提交的工作区）
lane gc --dry-run
lane gc
```

---

## 环境变量与端口契约

| 变量名 | 含义与范围 |
| :--- | :--- |
| `LANE_WORKSPACE` | 当前工作区绝对路径（容器内为 `/workspace`） |
| `LANE_ROOT` | 关联的主仓库根路径 |
| `LANE_DATA_DIR` | 工作区专属数据目录（原生下为 `.lane/data`，容器内为 `/workspace/.lane/data`） |
| `LANE_PORT_<NAME>` | **当前执行环境内**的服务监听端口（容器模式下对应内部监听端口） |
| `LANE_HOST_PORT_<NAME>`| **宿主机发布端口**（供宿主浏览器访问或外部调用使用） |

> [!NOTE]
> 发布端口契约支持 IPv4 Loopback (127.0.0.1) 上的 TCP 流量。详细网络和生命周期规范请参阅 [运行契约 (docs/runtime.md)](docs/runtime.md)。

---

## 常用命令速查

| 命令 | 说明 | 常用参数 |
| :--- | :--- | :--- |
| `lane init` | 初始化生成 `lane.yaml` 并安装 Agent skill | `--force` |
| `lane new <slug>` | 从 baseline 创建独立 Git worktree 工作区 | `--up`, `--base <branch>` |
| `lane adopt` | 将已存在的外部 Git worktree 纳入 lane 管理 | `--setup` |
| `lane attach` | 输出当前工作区的环境变量导出语句 | `--json` |
| `lane ls` | 列出所有工作区及其运行状态 | `--json` |
| `lane status` | 查看指定工作区的进程与探针状态 | `--json` |
| `lane ports` | 查看已分配的主机发布端口与监听端口 | `--json` |
| `lane plan` | 审查工作区运行契约与端口分配计划 | |
| `lane up` | 启动当前工作区声明的所有服务进程 | |
| `lane down` | 优雅停止当前工作区服务（保留数据） | |
| `lane logs <proc>` | 查看指定托管进程的标准输出与错误日志 | |
| `lane run -- <cmd>` | 在工作区执行上下文中运行一次性命令 | |
| `lane reset` | 重置私有数据目录并重新触发 setup hooks | |
| `lane done` | 安全销毁工作区及分支（要求代码已推送保存）| `--force` |
| `lane gc` | 垃圾回收过期、已被物理删除的残留登记 | `--dry-run`, `--json` |
| `lane doctor` | 检查并诊断运行环境依赖与状态健康度 | `--fix`, `--json` |
| `lane open <port>` | 在宿主默认浏览器中打开对应服务的发布地址 | `--json` |
| `lane skill install` | 将嵌入的 `SKILL.md` 重新安装到 Agent 目录 | |
| `lane hook install` | 安装 Cursor / Claude Code 的 worktree hook | `[cursor\|claude\|all]` |

---

## 开发与验证

项目遵循严苛的代码质量与幂等性保障规则，提交前须执行：

```sh
test -z "$(gofmt -l cmd internal)"
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
go test -race ./cmd/... ./internal/...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./cmd/... ./internal/...
go build ./cmd/lane
```

准备好固定版本的 process-compose（运行 `lane doctor --fix` 并将其加入 PATH）后，可运行端到端测试：
- 原生多工作区并发测试：`python tests/native_e2e.py ./lane`（Windows 使用 `./lane.exe`）
- 容器多工作区端到端测试：`python3 tests/container_e2e.py ./lane lane-runtime:local`

更多底层设计细节详见 [架构设计记录 (docs/architecture.md)](docs/architecture.md)。
