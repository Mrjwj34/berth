<p>
  <a href="https://github.com/Mrjwj34/lane">
    <img src="https://cdn.jwjbox.dev/lane.png" alt="lane logo" align="left" width="110" style="margin-right: 20px; margin-bottom: 12px;" />
  </a>
  <span style="font-size: 1.5em; font-weight: bold; line-height: 1.3;">快速、低成本地管理并行开发环境</span><br><br>
  <a href="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/Mrjwj34/lane/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/lane" alt="Latest Release" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/lane" alt="Go Version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/lane" alt="开源许可" /></a>
</p>
<br clear="left" />

<p align="center">
  <a href="README.md">English</a> | 简体中文
</p>

lane 为多个 Agent 并行编程快速拉起轻量、隔离的本地工作区，将 Git worktree、私有数据目录、动态端口分配以及受管进程融为一体，无需承担虚拟机的庞大开销。

<p align="center">
  <img src="https://cdn.jwjbox.dev/berth-demo.gif" alt="berth 演示" width="760" />
</p>

## 安装

你可以通过以下任意一种方式安装 lane。

### 下载预编译二进制

前往 GitHub Releases 页面下载对应操作系统的归档文件，解压后将可执行文件放入系统 PATH 路径。

### 使用 Go 安装

需要 Go 1.25 或更高版本：

```sh
go install github.com/Mrjwj34/lane/cmd/lane@latest
```

### 从源码构建

```sh
git clone https://github.com/Mrjwj34/lane.git
cd lane
go build -o bin/lane ./cmd/lane
```

## 30 秒快速开始

### 1. 初始化项目

在 Git 仓库根目录下执行：

```sh
lane init
lane hook install all
```

该操作会生成基础模板，把 Agent 技能安装到 `.agents/skills/lane`（Cursor、Codex、pi、Antigravity 均读取该目录），并把 Cursor 工作树适配器合并进 `.cursor/worktrees.json`。

### 2. 让 Agent 自动配置 lane.yaml

直接向你的编程助手发送提示词：

> 检查当前仓库，根据现有的启动脚本、环境依赖与监听端口，自动完成 lane.yaml 配置。

Agent 会自动读取内置的技能定义，分析项目文件并生成匹配的端口与进程定义。

如果倾向手动编写，请参阅详细功能文档中的配置指南与完整参考：[docs/features.md](docs/features.md)。

### 3. 创建独立工作区并启动服务

```sh
lane new feature-a --up
```

### 4. 在工作区上下文中执行测试

```sh
lane run -- npm test
```

### 5. 开发完成并推送提交后安全销毁

```sh
lane done feature-a
```

## 为什么存在

当多个 Agent 同时在同一个代码库上并行开发时，单纯切分支远远不够。只要同时运行测试套件或启动后台服务，立刻就会遭遇本地端口冲突、数据库状态污染以及孤儿进程泄漏。

以往开发者通常要在几种折中方案里痛苦权衡。纯 Git worktree 仅管理源码文件，对端口、数据库与后台进程完全放任不管；手动修改端口极易将本地临时配置误提交到远程仓库；而为每个工作区分裂一套完整的 Docker 虚拟机又极其消耗系统内存，且在非 Linux 宿主上存在显著的文件挂载性能损耗，拖慢 Agent 的反馈循环。

lane 针对这一痛点提供了一体化的工作区抽象。每个工作区同时拥有专属的 Git linked checkout、私有数据目录、原子端口分配与进程编排能力。所有操作毫秒级完成，并且完全不依赖常驻后台守护进程。

## 和 Docker、devcontainer 以及 Git worktree 的区别

### 单纯使用 Git worktree

Git worktree 只负责文件树和分支的隔离，端口分配、数据库持久化、后台进程生命周期以及环境变量注入全部处于未管理状态，需要开发者在工作区之外手动处理。

### Docker 与 devcontainer

Docker 与 devcontainer 提供完整的操作系统级虚拟化隔离。但它们伴随着显著的启动延迟、高内存占用、在 macOS 和 Windows 上的文件系统性能损耗，以及频繁重新构建镜像的开销。它们把整个虚拟机系统视作唯一的隔离边界。

### lane

lane 选择了更务实的中间路线。在原生模式下，它直接运行宿主机普通进程，零虚拟化损耗，同时自动分区端口、数据目录和环境变量。在容器模式下，它每个工作区复用单个 Linux 容器，无需每次重新构建镜像，也不拦截底层系统调用，通过回环网关转发流量。lane 将工作区而非整个虚拟机作为核心隔离单元。

## 架构

lane 将工作区生命周期管理与底层执行环境解耦。每个工作区独立维护分支配置、执行计划与操作锁。

<div align="center">
  <img src="https://cdn.jwjbox.dev/lane-architecture.png" alt="lane 架构图" width="800" />
</div>

详细文档导航：
- 用户指南：[docs/features.md](docs/features.md) 介绍完整配置参考、实用工作流以及 Agent 工具集成
- 运行时契约：[docs/runtime.md](docs/runtime.md) 详细说明网络模式、端口映射、容器运行方式与生命周期规则
- 架构设计记录：[docs/architecture.md](docs/architecture.md) 详细介绍无守护进程锁设计、状态恢复机制与安全不变式
