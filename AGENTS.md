# 仓库开发规范

`lane` 是一个项目无关、harness 无关、面向开源的本地 Agent 工作区轻量级 CLI 工具（Go 单二进制）。它通过 **Git worktree** 实现代码隔离，通过 **每工作区原生进程 + 唯一端口块 + 私有数据目录（`$LANE_DATA_DIR`）** 实现环境隔离，为 AI Agent 和人类开发者提供免虚拟机（Zero-VM）、秒级冷启动的本地多任务并行环境。

## 构建与检查

Go 版本以 `go.mod`（Go 1.25+）为准。在仓库根目录执行检查与构建：

```sh
# 格式化与静态检查
unformatted=$(gofmt -l cmd internal)
test -z "$unformatted" || { echo "Unformatted: $unformatted"; exit 1; }
go vet ./cmd/... ./internal/...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./cmd/... ./internal/...

# 测试与竞态检查
go test ./cmd/... ./internal/...
go test -race ./cmd/... ./internal/...

# 编译单二进制
go build -o bin/lane ./cmd/lane
```

也可使用 `just` 工具链完成日常研发：

```sh
just check     # 运行格式化、vet 与测试
just test      # 运行并发竞态测试 (-race)
just build     # 编译生成 bin/lane
just fmt       # 自动格式化 Go 代码
```

Go 检查范围限定在 `cmd` 和 `internal`，避免扫描非项目代码。

---

## 核心架构与设计约束

### 1. 原语与抽象边界
- **一切皆进程**：工具只提供通用隔离原语，**不认识任何具体后端或中间件组件**（Postgres、MySQL、Redis、Elasticsearch、SQLite 等均由项目在 `lane.yaml` 中通过进程命令或环境变量声明）。
- **工作区四要素**：
  1. **代码**：独立的 Git worktree。
  2. **端口**：独立的命名端口块（`LANE_PORT_<NAME>`），由机器级注册表保证唯一性并做 bind 探测。
  3. **存储**：私有数据目录 `LANE_DATA_DIR`（默认 `<worktree>/.lane/data`），用于承载数据库文件、缓存或运行时文件，随工作区删除而自动销毁。
  4. **进程**：由项目声明的原生进程集合，直通 process-compose 监督。
- **Harness 无关**：核心功能仅依赖 CLI + `lane.yaml` + `SKILL.md`。Cursor 的 `worktrees.json` 或 Claude Code 的 hooks 仅作为可选生成的适配层（`lane hook install`），绝不依赖任何具体编辑器或 Harness 的私有 SDK。
- **以目录为身份**：工具以目录路径作为工作区核心身份凭证，与外部工具（Cursor/Claude 手动创建或删除的 worktree）天然兼容，并能自动回收"目录已消失"的残留端口与后台进程。

### 2. 包职责划分
- `cmd/lane/`: 纯 CLI 交互层（cobra），负责解析命令行参数、组织输入输出（stdout/stderr）、格式化 `--json` 与退出码，严禁内嵌复杂业务逻辑。
- `internal/app`: 工作区生命周期编排（new/adopt/up/done/gc 等），组装各原语包。
- `internal/home`: `$LANE_HOME`（默认 `~/.lane`）路径解析，供测试隔离真实用户状态。
- `internal/config`: `lane.yaml` 解析、校验、默认值注入与 `${LANE_*}` 环境变量派生。
- `internal/worktree`: 封装 `git` worktree 创建、挂载、删除及 `.worktreeinclude` 规则拷贝。
- `internal/copyfs`: `copy_dirs` 的 clonefile / 硬链接 / 普通复制。
- `internal/ports`: 机器级端口块分配与 bind 探测。
- `internal/state`: 全局状态管理（`$LANE_HOME/state.json`），**必须使用文件锁（File Lock）与原子写入（Atomic Write）**，杜绝多 Agent 并发操作冲突。
- `internal/process`: 进程编排桥接，负责生成 `.lane/pc.yaml`，自动拉取钉死版本的 process-compose，并通过 `process-compose up -D -t=false` 托管进程。
- `internal/gc`: 垃圾回收状态机（已删除目录回收、空闲停机、已合入分支自动清理、工作区配额上限淘汰）。
- `internal/skill`: 内嵌 `SKILL.md` 与可选 harness hook 资产分发。
- `internal/gitx`: 带 `context.Context` 的 git 命令封装。

---

## 代码风格规范

- **命名与包设计**：沿用标准 Go 风格。小写单字包名，保持职责单一；不引入过度抽象的通用设计模式，优先满足当前里程碑需求。
- **错误处理**：
  - 错误必须向上传递给调用方，顶层统一输出。
  - 附加上下文信息时统一使用 `fmt.Errorf("...: %w", err)` 包装。
  - 使用 `errors.Is` / `errors.As` 进行错误判定，**严禁通过匹配错误字符串**来做逻辑判断。
  - 不使用 `panic` 处理业务流与预期系统错误。
- **并发与上下文控制**：
  - 所有涉及 I/O、执行外部命令（如 git/process-compose）、网络探针的函数，第一参数必须为 `ctx context.Context`。
  - 必须明确每个 goroutine 的生命周期、停止触发源与 channel 归属，杜绝后台泄漏。
- **Agent-First CLI 表面**：
  - 所有的查询和状态命令（`ls`, `status`, `ports`, `doctor` 等）必须提供 `--json` 标志，输出结构稳定、字段语义明确的 JSON。
  - **Actionable Error**：错误信息必须清晰指出问题原因，并给出下一步的可执行修复建议（例如："worktree is dirty. Commit changes or use --force"）。
  - 所有命令具备幂等性，重复执行安全。
- **注释与文档**：
  - 代码注释统一使用英文，只解释不明显的意图、不变量约束或跨平台细节。
  - 关键状态文件结构与配置格式在代码中声明明确的 Go struct 与 JSON/YAML tag。

---

## 测试与提交规范

- **测试设计**：
  - 测试文件命名为 `*_test.go`，与被测源码位于同一包内。
  - 涉及文件系统与 Git 操作的测试必须使用 `t.TempDir()` 创建独立的测试沙箱，严禁读写用户真实环境与全局 `~/.lane` 状态。
  - 针对并发写入或状态分配的逻辑，必须编写并发测试并保证 `go test -race` 通过。
- **Git 提交**：
  - 分支命名采用 `<type>/<slug>`（如 `feat/worktree-ls`, `fix/port-race`）。
  - 提交信息遵循 **Conventional Commits**（`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `ci:`, `chore:`）。
  - 行为或配置项变更必须在同一 PR 中同步更新受影响的文档与测试。
