# RULE.md — 仓库执行与提交约束规范

本仓库所有 AI Agent 与协作者在维护或扩展 **MyGo MD3E 原生组件库**（`github.com/imicola/mygo-md3e`）时，必须严格遵守以下不可豁免的工程纪律。

> 注：早期建库阶段的 `plan.md` 任务账本已完成使命并归档移除；当前项目已作为公开 Go 模块交付演进，所有开发、修复与重构均以模块边界、规格文档与自动化门禁为准。

---

## 1. 核心工程纪律：单主题收敛与基线同步

1. **单主题小步提交（One Concern, One Commit）**：每次改动只聚焦一个明确职责或问题域（如单个组件缺陷修复、性能/DCE 优化、文档示例同步、单组测试或跨平台基线补齐），**严禁把多个无关改动囤积成一个大 commit**。
2. **开工前同步与净基线检查**：开始任何新工作前，必须先检查 `git status --porcelain`（应为空）、`git log -n 5 --oneline`，并同步远端分支（如 `git fetch --prune`、`git status -sb`），确认基线干净且未落后于远端。
3. **保持每笔提交独立可用**：任何历史 commit 检出后都必须能独立通过全量编译与测试，确保可随时 `git revert` 或 `git bisect`。

---

## 2. 强制提交前门禁（Pre-Commit Gate）

在执行 `git commit` 之前，**必须在仓库根目录真实运行并亲眼观察以下门禁全部通过**（不得凭记忆断言、不得跳过、不得删减断言）：

1. **纯 Go 编译门禁（无 cgo）**
   ```sh
   CGO_ENABLED=0 go build ./...
   ```
   *(Windows PowerShell 下先设置 `$env:CGO_ENABLED="0"`。)*
2. **静态分析门禁**
   ```sh
   CGO_ENABLED=0 go vet ./...
   ```
3. **MyGo UI 生命周期静态检查门禁**（**注意：`mygo vet` 不带 `./...` 参数**）
   ```sh
   go tool mygo vet
   ```
4. **全量无缓存测试门禁**
   ```sh
   CGO_ENABLED=0 go test -count=1 ./...
   ```
   - 涉及交互、动效、主题、无障碍、RTL 或自定义绘制时，须在对应测试中覆盖：`ui.NewTester` 虚拟时钟推进与收敛（`tt.FramePending() == false`）、`ReduceMotion` 零多余帧、`ui.RightToLeft` 镜像、无障碍树角色/名称/状态、边界与非有限数值防护。
   - **Golden 基线纪律**：`golden_test.go` 仅锁定无文字纯几何渲染；日常测试不得改动或残留 `*.got.png`；仅在有意修改视觉输出时才使用 `MYGO_UPDATE_GOLDEN=1` 更新，且必须同时确保 Windows 默认基线（`testdata/<name>.png`）与已有非 Windows 平台基线（`testdata/<name>_<GOOS>.png`）同步有效。
5. **跨平台构建、独立示例与按需裁剪门禁**（涉及公共 API、平台相关逻辑、示例或基础设施改动时必跑）
   - 跨平台交叉编译：
     ```sh
     GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build ./...
     GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
     GOOS=js      GOARCH=wasm  go build ./...
     ```
   - Gallery 与独立子模块 `examples/m2-demo` 验证（构建可执行文件时使用 `-o NUL` / `-o /dev/null` 或立即清理产物）：
     ```sh
     CGO_ENABLED=0 go build -o NUL ./examples/gallery
     cd examples/m2-demo && CGO_ENABLED=0 go vet ./... && go tool mygo vet && CGO_ENABLED=0 go test -count=1 ./... && cd ../..
     ```
   - 若改动涉及包级状态、`sync.Once`、全局表或核心基础设施（`tokens/`、`motion/`、`shape/`、`theme.go`、`surface.go`、`ripple.go`、`loading.go`、`internal/mdicon/` 等），须确认不会破坏 **Zero-Bloat 按需链接**（仅引入 `Button` 的最小程序不应链接进未使用的重型组件或 35 形状全表）。

> **禁止事项**：
> - 严禁提交无法编译或任一测试失败的代码。
> - 严禁通过删除测试、放宽容差/断言、`t.Skip` 或注释报错代码来“凑绿”。
> - 严禁在未通过门禁时使用 `--no-verify` 强行提交。

---

## 3. 提交信息格式与文档同步

1. **强制 Commit Message 规范**（遵循 Conventional Commits）：
   ```text
   <type>(<scope>): <简明改动说明>

   - 改动摘要：<核心新增/修改的文件、原因与职责>
   - 验收结果：<列出实际执行的门禁命令与 PASS 结果>
   ```
   - `<type>` 仅限：`feat`、`fix`、`refactor`、`test`、`docs`、`perf`、`chore`。
   - `<scope>` 为受影响包名或组件域（如 `button`、`navigation`、`picker`、`tokens`、`motion`、`shape`、`theme`、`golden`、`gallery`、`docs`、`repo`）。
   - 不再依赖或强制要求已移除的 `plan.md` 任务点编号（如 `[T1.1]`）。
2. **文档与公开示例同步**：修改公开 API 签名、`Options` 字段、主题/无障碍/RTL 行为或构建命令时，必须同步更新 GoDoc 注释/示例（外部示例一律使用 `md3e.IconDraw` 回调，禁止引导引用 `internal/mdicon`）、[README.md](README.md) / [README.zh-CN.md](README.zh-CN.md)、[docs/architecture.md](docs/architecture.md) 与 [docs/md3e-spec.md](docs/md3e-spec.md)。
3. **禁止 CI 跳过标记**：严禁在 commit message 的标题或正文中出现 `[skip ci]`、`[ci skip]`、`[no ci]`。

---

## 4. 边界与红线约束

1. **上游 `github.com/egoist/mygo` 只读边界与模块规范**：
   - 根 `go.mod` 直接依赖上游公开模块版本，**严禁将指向本地路径的 `replace`（如 `replace github.com/egoist/mygo => ./mygo`）提交进 `go.mod`**（`examples/m2-demo/go.mod` 指向仓库根目录 `../..` 除外）。
   - 若工作区存在 `.gitignore` 忽略的本地 `mygo/` 目录，仅作只读参考，严禁修改 `mygo/` 下任何源码；严禁导入 `github.com/egoist/mygo/internal/...`。
2. **严禁提交临时文件与失败产物**：禁止提交 `*.exe`、`*.test`、`*.out`、`*.got.png`、`build/`、`.mygo/`、`.screenshots/` 等产物；提交前须用 `git status` 和 `git diff --cached` 核对暂存文件。
3. **禁止破坏已冻结设计契约**：未经明确需求，不得擅自更改 `tokens` 语义角色、12 档弹簧阻尼比换算关系或破坏既有公开函数签名。

---

## 5. 遇阻处理协议（Stop & Escalate）

如果某项工作连续尝试两次仍因以下原因无法满足门禁：
- 发现上游 `mygo/ui` 公开 API 缺失无法绕过的原语；
- 需求规格存在歧义或互相冲突；
- 环境/工具链存在无法自行修复的异常；

则 **立即停止推进并禁止强行提交半成品或绕过约束**，保持可追溯状态并向用户清晰汇报：当前卡点、已尝试方案、阻塞原因与精确复现命令。
