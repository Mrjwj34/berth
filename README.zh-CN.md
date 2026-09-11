<div align="center">
  <img src="https://cdn.jwjbox.dev/lane.png" alt="lane logo" width="120" />
  <h1>lane</h1>
  <p>快速、低成本地管理并行开发环境</p>
  <p>
    <a href="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
    <a href="https://github.com/Mrjwj34/lane/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/lane" alt="Latest Release" /></a>
    <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/lane" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/lane" alt="License" /></a>
  </p>
  <p>
    简体中文 | <a href="README.md">English</a>
  </p>
</div>

## 功能特性

- 基于 Git worktree 提供完全独立的工作区，各自拥有私有数据目录、配置与独立端口分配
- 原生执行模式直接运行宿主机普通进程，配合动态端口分配与进程就绪探针
- 容器执行模式每个工作区复用单个 Linux 容器，通过回环转发保留硬编码监听端口
- 无后台常驻守护进程，基于局部文件锁与状态持久化保证跨进程安全
- 严格的生命周期防误删机制，阻断删除主仓库，保护未推送提交与外部接管的工作区
- 内置 Agent 技能，支持一键安装到 Claude Code 和 Cursor 等编程助手

## 架构

lane 将工作区生命周期管理与底层执行环境解耦。每个工作区独立维护分支配置、执行计划与操作锁。

<div align="center">
  <img src="https://cdn.jwjbox.dev/lane-architecture.png" alt="lane 架构图" width="800" />
</div>

## 快速开始

### 安装方式

下载预编译二进制包：

从 GitHub Releases 页面下载对应操作系统的归档文件，解压后将可执行文件放入系统 PATH 路径。

使用 Go 安装：

```sh
go install github.com/Mrjwj34/lane/cmd/lane@latest
```

从源码构建：

```sh
git clone https://github.com/Mrjwj34/lane.git
cd lane
go build -o bin/lane ./cmd/lane
```

### 初始化项目

在 Git 仓库根目录下执行初始化命令：

```sh
lane init
```

该命令会生成默认的 lane.yaml 文件，并将技能定义安装到 Agent 对应目录。如果使用 Cursor 或 Claude Code 工作区集成，可以安装钩子脚本：

```sh
lane hook install all
```

### 配置 lane.yaml

声明基线分支、端口需求以及受管进程：

```yaml
version: 1
base: main
ports: [web]
env:
  PORT: ${LANE_PORT_WEB}
processes:
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get: {host: 127.0.0.1, port: "${LANE_PORT_WEB}", path: /}
```

### 管理工作区

创建并启动新工作区：

```sh
lane new feature-a --up
```

查看当前工作区列表与分配的端口：

```sh
lane ls --json
lane status feature-a
```

在当前工作区环境中执行单次测试或命令：

```sh
lane run -- npm test
```

暂停并停止当前工作区服务：

```sh
lane down feature-a
```

提交或合并改动后安全销毁工作区：

```sh
lane done feature-a
```

## 详细文档

- 运行时契约：[docs/runtime.md](docs/runtime.md) 详细说明网络模式、端口映射、容器运行方式与生命周期规则
- 架构设计记录：[docs/architecture.md](docs/architecture.md) 详细介绍无守护进程锁设计、状态恢复机制与安全不变式
