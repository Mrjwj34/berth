<p>
  <a href="https://github.com/Mrjwj34/berth">
    <img src="https://cdn.jwjbox.dev/berth.png" alt="berth logo" align="left" width="110" style="margin-right: 20px; margin-bottom: 12px;" />
  </a>
  <span style="font-size: 1.5em; font-weight: bold; line-height: 1.3;">快速、低成本地管理并行开发环境</span><br><br>
  <a href="https://github.com/Mrjwj34/berth/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/berth/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/Mrjwj34/berth/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/berth" alt="Latest Release" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/berth" alt="Go Version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/berth" alt="开源许可" /></a>
  <a href="https://mrjwj34.github.io/berth/"><img src="https://img.shields.io/badge/docs-website-blue" alt="文档站" /></a>
</p>
<br clear="left" />

<p align="center">
  <a href="README.md">English</a> | 简体中文
</p>

berth 为多个 Agent 并行编程快速拉起轻量、隔离的本地工作区，将 Git worktree、私有数据目录、动态端口分配以及受管进程融为一体，无需承担虚拟机的庞大开销。每个工作区都是一个自洽的开发环境——独立分支、独立数据、独立端口、独立受管进程——毫秒级创建，且不依赖常驻后台服务。

<p align="center">
  <img src="https://cdn.jwjbox.dev/berth-demo.gif" alt="berth 创建独立工作区并启动声明的服务" width="760" />
</p>

## 安装

你可以通过以下任意一种方式安装 berth。

### Homebrew（macOS 与 Linux）

```sh
brew install Mrjwj34/tap/berth
```

### Scoop（Windows）

```powershell
scoop bucket add berth https://github.com/Mrjwj34/scoop-bucket
scoop install berth
```

### 下载预编译二进制

前往 GitHub Releases 页面下载对应操作系统的归档文件，解压后将可执行文件放入系统 PATH 路径。

### 使用 Go 安装

需要 Go 1.25 或更高版本：

```sh
go install github.com/Mrjwj34/berth/cmd/berth@latest
```

### 从源码构建

```sh
git clone https://github.com/Mrjwj34/berth.git
cd berth
go build -o bin/berth ./cmd/berth
```

## 30 秒快速开始

### 1. 初始化项目

在 Git 仓库根目录下执行：

```sh
berth init
berth hook install all
```

该操作会生成基础模板，把 Agent 技能安装到 `.agents/skills/berth`（Cursor、Codex、pi、Antigravity 均读取该目录），并把 Cursor 工作树适配器合并进 `.cursor/worktrees.json`。

### 2. 让 Agent 自动配置 berth.yaml

直接向你的编程助手发送提示词：

> 检查当前仓库，根据现有的启动脚本、环境依赖与监听端口，自动完成 berth.yaml 配置。

Agent 会自动读取内置的技能定义，分析项目文件并生成匹配的端口与进程定义。

如果倾向手动编写，请参阅详细功能文档中的配置指南与完整参考：[docs/features.md](docs/features.md)。

### 3. 创建独立工作区并启动服务

```sh
berth new feature-a --up
```

### 4. 在工作区上下文中执行测试

```sh
berth run -- npm test
```

### 5. 开发完成并推送提交后安全销毁

```sh
berth done feature-a
```

## 为什么并行的 Agent 会互相冲突

当多个 Agent 同时在同一个代码库上并行开发时，单纯切分支远远不够。只要同时运行测试套件或启动后台服务，立刻就会遭遇本地端口冲突、数据库状态污染以及孤儿进程泄漏。

以往开发者通常要在几种折中方案里痛苦权衡。纯 Git worktree 仅管理源码文件，对端口、数据库与后台进程完全放任不管；手动修改端口极易将本地临时配置误提交到远程仓库；而为每个工作区分裂一套完整的 Docker 虚拟机又极其消耗系统内存，且在非 Linux 宿主上存在显著的文件挂载性能损耗，拖慢 Agent 的反馈循环。

berth 针对这一痛点提供了一体化的工作区抽象。每个工作区同时拥有专属的 Git linked checkout、私有数据目录、原子端口分配与进程编排能力。所有操作毫秒级完成，并且完全不依赖常驻后台守护进程。

## 和 git worktree、Docker、devcontainer 的对比

| | 纯 `git worktree` | devcontainer / Docker | **berth** |
| --- | --- | --- | --- |
| 隔离边界 | 文件与分支 | 整台机器 / 虚拟机 | 工作区：worktree、私有数据、端口 |
| 创建新环境 | 瞬时，仅源码文件 | 构建镜像并启动容器 | 毫秒级，无守护进程 |
| 端口分配 | 手动修改，易被误提交 | 容器网络 | 原子预留，`BERTH_PORT_*` |
| 数据隔离 | 无 | volume 与 bind mount | 每个工作区独立 `.berth/data` |
| 进程管理 | 无 | 容器内部 | process-compose 或容器运行时 |
| 宿主文件系统性能 | 原生 | macOS / Windows 上有挂载损耗 | 原生，支持时使用写时复制 |

### 单纯使用 Git worktree

Git worktree 只负责文件树和分支的隔离，端口分配、数据库持久化、后台进程生命周期以及环境变量注入全部处于未管理状态，需要开发者在工作区之外手动处理。

### Docker 与 devcontainer

Docker 与 devcontainer 提供完整的操作系统级虚拟化隔离。但它们伴随着显著的启动延迟、高内存占用、在 macOS 和 Windows 上的文件系统性能损耗，以及频繁重新构建镜像的开销。它们把整个虚拟机系统视作唯一的隔离边界。当你确实需要不同的内核、完整的系统隔离，或宿主无法提供某套工具链时，它们才是正确选择。

### berth

berth 选择了更务实的中间路线。在原生模式下，它直接运行宿主机普通进程，零虚拟化损耗，同时自动分区端口、数据目录和环境变量。在容器模式下，它每个工作区复用单个 Linux 容器，无需每次重新构建镜像，也不拦截底层系统调用，通过回环网关转发流量。berth 将工作区而非整个虚拟机作为核心隔离单元。

## Agent 集成

berth 就是一条普通 CLI，任何能执行 shell 命令的 Agent 都可以驱动它。此外 `berth init` 会把技能安装到唯一位置 `.agents/skills/berth/SKILL.md`，Cursor、Codex、pi 与 Antigravity 均读取该目录，因此只需维护一份副本；berth 不会向 `AGENTS.md`、`GEMINI.md` 或任何宿主工具自己的规则文件写入内容。

`berth hook install cursor` 会把 Cursor 工作树适配器合并进 `.cursor/worktrees.json`，使 Cursor 在 Agents Window、IDE 或 CLI 中创建的每个工作树都会自动执行 `berth adopt --setup`。会话结束钩子是有意不安装的：请显式用 `berth down`（保留数据）或 `berth done`（释放工作区）收尾，被遗忘的工作区交给 `berth gc` 回收。

## 常见问题

**两个 Agent 能同时改同一个仓库吗？**

可以。每个工作区都是独立的 Git linked worktree，位于仓库旁的 `<repo>.berths/<slug>`，分支名为 `berth/<slug>`，拥有独立的私有数据目录与独立分配的端口。主检出永远不会成为工作区，因此在 Agent 并行工作期间依然可以正常使用。

**怎么让每个 Agent 用不同的端口？**

在 `berth.yaml` 中声明项目需要的命名端口，例如 `ports: [web, pg]`。berth 会在一次跨进程的原子事务中为每个工作区预留唯一宿主端口，且该分配在工作区整个生命周期内保持稳定。程序通过 `BERTH_PORT_WEB` 读取内部端口，通过 `BERTH_HOST_PORT_WEB` 从宿主浏览器访问服务。

**为什么不直接用 devcontainer 或 Docker？**

当你需要不同内核、完整系统隔离，或宿主无法提供的工具链时，就该用它们。代价是启动延迟、内存占用、macOS 与 Windows 上的挂载性能损耗，以及反复重建镜像的开销。berth 的原生模式直接运行宿主进程，零虚拟化损耗；容器模式则面向硬编码回环端口的项目——每个工作区复用单个 Linux 容器，不做任何按次镜像构建。

**berth 需要常驻守护进程吗？**

不需要。berth 依靠文件锁和一份机器级注册表文件工作。长耗时操作持有的是单个工作区的锁，而非全局机器锁；进程编排交由 process-compose 或容器运行时负责。

**工作区销毁后我的数据会怎样？**

`berth down <slug>` 会优雅停止进程并保留全部数据；`berth done <slug>` 会先校验工作成果已经被保存（已提交/已推送/已合并），再删除工作区及其 worktree；`berth reset <slug>` 只清空私有数据目录并重跑 setup 钩子；`berth gc` 回收被遗忘的工作区，且永不使用强制删除。

**支持 Windows 和 macOS 吗？**

支持。官方为 Linux、macOS、Windows 的 amd64 与 arm64 都发布原生二进制。容器模式需要本机已安装 Docker 或 Podman。可写依赖的复制在支持的文件系统上使用写时复制——Linux btrfs/xfs 用 reflink，APFS 用 clonefile——并且永远不会静默退化成硬链接。

## 架构

berth 将工作区生命周期管理与底层执行环境解耦。每个工作区独立维护分支配置、执行计划与操作锁。

<div align="center">
  <img src="https://cdn.jwjbox.dev/berth-architecture.png" alt="berth 架构图" width="800" />
</div>

详细文档导航：
- 文档站：[mrjwj34.github.io/berth](https://mrjwj34.github.io/berth/) 是渲染后的完整指南，包含三篇对比页
- 用户指南：[docs/features.md](docs/features.md) 介绍完整配置参考、实用工作流以及 Agent 工具集成
- 运行时契约：[docs/runtime.md](docs/runtime.md) 详细说明网络模式、端口映射、容器运行方式与生命周期规则
- 架构设计记录：[docs/architecture.md](docs/architecture.md) 详细介绍无守护进程锁设计、状态恢复机制与安全不变式
