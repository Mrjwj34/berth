<table>
  <tr>
    <td width="140" align="center" valign="middle">
      <img src="https://cdn.jwjbox.dev/lane.png" alt="lane logo" width="120" />
    </td>
    <td valign="middle">
      <p><strong>快速、低成本地管理并行开发环境</strong></p>
      <p>
        <a href="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
        <a href="https://github.com/Mrjwj34/lane/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/lane" alt="Latest Release" /></a>
        <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/lane" alt="Go Version" /></a>
        <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/lane" alt="License" /></a>
      </p>
      <p>
        简体中文 | <a href="README.md">English</a>
      </p>
    </td>
  </tr>
</table>

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

### 1. 安装 lane

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

### 2. 初始化项目

在 Git 仓库根目录下执行：

```sh
lane init
lane hook install all
```

此操作会生成基础模板，自动将技能定义分发至 Claude Code 与 Cursor 目录，并配置工作区生命周期钩子。

### 3. 让 Agent 自动配置工作区

直接向你的编程助手发送提示词：

> 检查当前仓库，根据现有的启动脚本、环境依赖与监听端口，自动完成 lane.yaml 配置。

Agent 会自动读取内置的技能定义，分析项目文件并生成匹配的端口与进程定义。

如果选择手动配置，可以直接编辑项目根目录下的 lane.yaml 文件：

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

### 4. 创建与运行工作区

可以让 Agent 直接创建环境，也可以在终端执行：

```sh
lane new feature-a --up
```

查看当前工作区列表与状态：

```sh
lane ls --json
lane status feature-a
```

在当前工作区环境中执行测试或命令：

```sh
lane run -- npm test
```

开发完成并将提交推送保存后，安全销毁工作区：

```sh
lane done feature-a
```

## 详细文档

- 用户指南：[docs/features.md](docs/features.md) 介绍具体的工作流、数据隔离策略以及 Agent 工具集成
- 运行时契约：[docs/runtime.md](docs/runtime.md) 详细说明网络模式、端口映射、容器运行方式与生命周期规则
- 架构设计记录：[docs/architecture.md](docs/architecture.md) 详细介绍无守护进程锁设计、状态恢复机制与安全不变式
