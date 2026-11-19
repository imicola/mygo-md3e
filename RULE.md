# RULE.md — 仓库执行与提交约束规范

本仓库所有 AI Agent 与协作者在开发 **MyGo MD3E 原生组件库** 时，必须严格遵守以下不可豁免的工程纪律，以确保每次推进都可追踪、可验证、可回滚。

---

## 1. 核心铁律：推进即提交（One Point, One Commit）

1. **按点推进**：所有开发工作必须严格按照 [plan.md](plan.md) 中的任务点编号（如 `T0.1`、`T1.1`、`T1.2`……）逐点推进，**严禁跳点**，除非阻塞前置点已被明确豁免。
2. **一点一提交**：每完成一个任务点并通过全部门禁验收后，**必须立即提交一次独立的 `git commit`**，然后才能开始下一个任务点。
3. **禁止囤积改动**：严禁把多个任务点合并成一个大 commit，也严禁在前一个点未提交的情况下继续开发下一个点。
4. **开工前工作区必须干净**：开始任何新点之前，必须先检查 `git status --porcelain`（应为空）与 `git log -n 5 --oneline`，确认基线干净且明确当前已推进到的位置。

---

## 2. 强制提交前四重门禁（Pre-Commit Gate）

在执行 `git commit` 之前，**必须在仓库根目录真实运行并亲眼观察以下命令全部通过**（不得凭记忆断言、不得跳过、不得删减断言）：

1. **纯 Go 编译门禁（无 cgo）**
   ```sh
   CGO_ENABLED=0 go build ./...
   ```
2. **静态分析门禁**
   ```sh
   CGO_ENABLED=0 go vet ./...
   ```
3. **MyGo UI 生命周期静态检查门禁**（自 `T0.1` 初始化 `go.mod` 起强制执行）
   ```sh
   go tool mygo vet ./...
   ```
4. **全量与本点专项测试门禁**
   ```sh
   CGO_ENABLED=0 go test ./...
   ```
   以及该点在 `plan.md` 中列出的专项验收（如跨平台构建检查、`ui.NewTester` 交互/动画/无障碍检查、像素/几何断言等）。

> **禁止事项**：
> - 严禁提交无法编译或任一测试失败的代码。
> - 严禁通过删除测试、放宽容差/断言、`t.Skip` 或注释报错代码来“凑绿”。
> - 严禁在未通过门禁时使用 `--no-verify` 强行提交。

---

## 3. 提交信息格式与进度账本同步

1. **进度账本同步**：在通过门禁后、执行 `git commit` **之前**，在同一次提交内将 `plan.md` 中对应任务点的 `- [ ]` 更新为 `- [x]`（禁止把 `plan.md` 状态更新留到下一个 commit 或遗漏）。
2. **强制 Commit Message 规范**：
   ```text
   <type>(<scope>): [<TaskID>] <简明改动说明>

   - 改动摘要：<核心新增/修改的文件与职责>
   - 验收结果：<列出实际执行的命令与 PASS 结果>
   ```
   - `<type>` 仅限：`feat`、`fix`、`refactor`、`test`、`docs`、`perf`、`chore`。
   - `<TaskID>` 必须精确对应 `plan.md` 中的点编号，例如 `feat(tokens): [T1.1] implement HCT color space and tonal conversions`。
   - 额外修复（非新功能点）须使用 `fix(<scope>): [<TaskID>-followup] ...` 或明确说明关联点。
3. **禁止 CI 跳过标记**：严禁在 commit message 的标题或正文中出现 `[skip ci]`、`[ci skip]`、`[no ci]`。

---

## 4. 边界与红线约束

1. **严禁修改 `mygo/` 目录**：`mygo/` 是上游 MyGo 框架的本地只读参考与 `replace` 目标（已被 `.gitignore` 忽略）。本组件库只能调用 `github.com/egoist/mygo` 的公开导出 API（`ui`、`ui/shader`、`ui/golden`、`mygo` 等），严禁修改 `mygo/` 下源码或导入 `internal/*`。
2. **严禁提交临时文件与失败产物**：禁止提交 `*.exe`、`*.test`、`*.got.png`、`build/`、`.mygo/` 等产物；提交前须用 `git status` 和 `git diff --cached` 核对暂存文件。
3. **保持每笔提交独立可用**：任何历史 commit 检出后都必须能通过 `CGO_ENABLED=0 go test ./...`，确保可随时 `git revert` 或 `git bisect`。

---

## 5. 遇阻处理协议（Stop & Escalate）

如果某个任务点连续尝试两次仍因以下原因无法满足门禁：
- 发现上游 `mygo/ui` 公开 API 缺失无法绕过的原语；
- 需求规格存在歧义或互相冲突；
- 环境/工具链存在无法自行修复的异常；

则 **立即停止推进该点及后续点**，**禁止强行提交半成品或绕过约束**，保持可追溯状态并向用户清晰汇报：当前卡点、已尝试方案、阻塞原因与精确复现命令。
