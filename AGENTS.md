# AGENTS.md — 开发者与 Coding Agent 工程指南

本项目是基于 **MyGo** 原生 GPU UI 工具包（`github.com/egoist/mygo/ui`）构建的 **Material Design 3 Expressive (MD3E)** 原生组件库，纯 Go 实现，零 HTML/WebView 依赖，支持桌面端（macOS / Windows / Linux）与浏览器（`js/wasm`）无头及真窗口运行。

---

## 1. 必读文档顺序（每次开工前阅读）

1. [RULE.md](RULE.md) — **最高优先级的提交、门禁与防越界约束**
2. [AGENTS.md](AGENTS.md) — 本文件：MyGo `ui` 硬技术规范与避坑指南
3. [docs/architecture.md](docs/architecture.md) — 模块分层、包结构、API 设计规范
4. [docs/md3e-spec.md](docs/md3e-spec.md) — 精确的 MD3E 色彩、动效、形状、排版与组件规格
5. [plan.md](plan.md) — 里程碑路线图、原子任务点清单与验收标准

如需查阅上游框架细节，请直接检索只读参考目录 `mygo/docs/ui/`（如 `views.md`、`motion.md`、`transitions.md`、`custom-widgets.md`、`testing.md`、`accessibility.md`、`styling.md`）与 `mygo/ui/` 源码注释。

---

## 2. 技术硬约束（来自 MyGo 框架源码验证）

### 2.1 纯 Go 与零内部依赖
- **禁止 cgo**：桌面端及浏览器目标下全部代码必须在 `CGO_ENABLED=0` 下通过编译与测试，禁止 `import "C"`。
- **禁止导入上游内部包**：只能导入 `github.com/egoist/mygo` 及其公开子包（如 `ui`、`ui/shader`、`ui/golden`），严禁引用 `github.com/egoist/mygo/internal/...`。
- **禁止修改 `mygo/`**：不得更改 `mygo/` 目录下的任何文件。
- **最小化外部依赖**：除标准库与 `github.com/egoist/mygo` 外，不引入多余第三方 Go 依赖。
- **按需链接（Zero-Bloat）**：禁止在包级 `init()` 中全局注册所有组件或引入大体积全局表；保证只使用 `Button` 的应用不会被链接进未使用的重型组件或图标数据。

### 2.2 `ui.Element` 与 `*ui.Context` 生命周期铁律
MyGo `ui` 采用受检的值句柄与单帧多 Pass 竞技场分配器（见 `mygo/docs/ui/views.md` 与 `mygo/docs/ui/migration.md`）：
- **绝不持久化保存**：`ui.Element`、`*ui.Context` 及 `ui.*Parts`（如 `SegmentedParts`、`TabsParts`、`SplitButtonParts`、`SelectParts[T]` 等）仅在当前一次 build pass 内有效，**严禁存入 struct 字段、全局变量、跨帧闭包或传给 goroutine**。
- **持久身份与跨帧状态**：
  - 业务状态由调用方持有并通过指针传入（如 `*bool`、`*float64`、`*int`、`*string`），或用 `ui.Local(e, key, init)` 绑定到元素持久状态。
  - 跨帧控件身份与焦点控制必须用 `ui.Handle`（`.Bind(&handle)`），后台回调/重绘用 `c.Services()`。
  - 所有生命周期违规会被 `go tool mygo vet ./...` 与 `ui.NewTester` 直接拦截报错/panic。
- **Key 赋值的严格顺序**：
  - 有状态内置控件与 Base（如 `CheckboxBase`、`SwitchBase`、`SliderBase`、`TabsBase`、`SegmentedBase`、`SelectBase`、`TextInputBase`）**必须在调用构造函数之前**用 `c.Key(k)` 指定 key（例如 `ui.SwitchBase(c.Key("wifi"), on)`）。
  - 容器元素若调用 `e.Key(k)`，必须在调用 `ui.Local`、读取 `Clicked()`/`Hovered()`/`Focused()` 或构建子元素之前完成。
  - 同一父节点下兄弟元素严禁出现重复 key。

### 2.3 输入模型、重绘性能与绘制原则
- **构建即应用输入**：MyGo 的控件在构造函数返回时即完成用户输入写入并触发同帧二遍重建；`Changed()`/`Submitted()` 读取该变更。不要在 view 中缓存派生局部副本覆盖用户输入而不回写。
- **避免 hover/press 引起整树无谓重建**：在 view 构建阶段直接调用 `e.Hovered()` 或 `e.Pressed()` 会导致指针悬停/离开时整棵 UI 树重建（见 `mygo/docs/ui/performance.md`）。
  - 对于**纯视觉变化**（如 MD3E StateLayer 透明度叠加、Ripple 扩散、纯色阴影变化），优先在 `e.Draw` / `e.DrawOver` 回调或 `ui.Local` 中读取 `e.Hovered()` / `e.Pressed()` 并通过 `p.After` / `p.AnimationFrame` 驱动重绘。
  - 对于必须参与布局或元素属性（如 MD3E 按钮按压圆角形变、ButtonGroup 邻居宽度挤压）的动画，可在构建阶段配合 `e.AnimateTo` 使用。

### 2.4 弹簧物理公式关键陷阱（必读！）
上游 `mygo/ui/motion.go` 中 `ui.SpringOptions{Stiffness, Damping}` 的字段含义如下：
- `Mass` 默认为 `1`。
- **`Damping` 是物理阻尼系数 $c$，而不是无量纲阻尼比 $\zeta$（damping ratio）！**
- Material 3 Expressive 规范给出的是 `(dampingRatio ζ, stiffness k)`。转换公式为：
  $$c = 2 \cdot \zeta \cdot \sqrt{k \cdot m}$$
- 若直接把 $\zeta = 0.6$ 写入 `Damping: 0.6`，将导致严重欠阻尼震荡！因此 **所有 MD3E 弹簧动画必须统一调用 `motion` 包封装好的构造函数**，严禁组件层手写裸 `ui.SpringOptions`。

### 2.5 圆角、路径、偏好适配与无障碍
- **独立四角圆角**：`ui.Element.Radius(tl, tr, br, bl)` 原生支持 4 个独立圆角半径（1 参为全角，4 参为顺时针左上、右上、右下、左下）。
- **复杂形变形状**：MD3E 35 种内置形状与连续 Morph 必须通过 `ui.Path`（`MoveTo`/`LineTo`/`CubeTo`/`Close`）配合 `Painter.FillPath` / `StrokePath` / `ClipPath` 绘制。
- **无障碍（Accessibility）**：所有交互组件必须构建在对应的 `ui.*Base` 之上，或显式设置 `Role`、`Label`、`Checked`/`AccessSelected`/`Expanded`/`Range`/`Step`/`Disabled`/`AccessDisabled`，保证 Tab/方向键/Space/Enter/Escape 键盘完整可用，且可被 `ui.Tester.AccessByRole` 检索。
- **ReduceMotion**：所有组件动画、涟漪、波浪进度条与形状加载器必须响应 `c.Preferences().ReduceMotion`。开启减少动态效果时，位置/大小/形态直接收敛到目标态且**不再请求持续动画帧**（`tt.FramePending() == false`）。
- **RTL（从右到左）**：布局边距与锚点优先使用 `PaddingStart`、`PaddingEnd`、`MarginStart`、`MarginEnd`、`Start`、`End`、`AnchorTopStart` 等逻辑方法；进度条、滑块、导航指示器与方向图标需正确处理 `c.Direction() == ui.RightToLeft`。

### 2.6 测试规范
- **严禁使用 `time.Sleep` 或真实系统时钟等待**：全部 UI 测试使用 `tt := ui.NewTester(...)` 配合 `tt.Advance(d)` 推进虚拟时钟。
- **必须覆盖的核心测试维度**：
  1. 交互状态转换（点击、拖动、键盘导航、快捷键、禁用态拦截）；
  2. 弹簧与动效收敛（推进足够时间后 `tt.FramePending() == false`）；
  3. `ReduceMotion` 行为（无多余动画帧、形态正确）；
  4. 无障碍树验证（`tt.AccessByRole` / `tt.AccessAll` 的角色、名称、值、状态）；
  5. 像素/几何边界检查（对自定义绘制组件如 Shape、Morph、WavyProgress、LoadingIndicator、Ripple 进行像素包围盒与中心点颜色采样断言）。

---

## 3. 本机工具链说明

- 目标 Go 版本：`go 1.27.3`（与 `mygo/go.mod` 对齐）。
- Windows 下无 cgo 工具链时，禁止执行 `-race`（`go test -race` 依赖 cgo），统一使用 `CGO_ENABLED=0` 执行构建、测试与跨平台交叉编译验证。
- 常用验证命令速查：
  ```sh
  CGO_ENABLED=0 go build ./...
  CGO_ENABLED=0 go vet ./...
  go tool mygo vet ./...
  CGO_ENABLED=0 go test ./...
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...
  GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./...
  GOOS=js GOARCH=wasm go build ./...
  ```
