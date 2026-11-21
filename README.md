# mygo-md3e

**MyGo MD3E** 是构建在 [MyGo](https://github.com/egoist/mygo) 原生 GPU UI 工具包之上的 **Material Design 3 Expressive (MD3E)** 组件库：纯 Go 实现，零 cgo，零 HTML/WebView 依赖，桌面端（macOS / Windows / Linux）与浏览器（`GOOS=js GOARCH=wasm`）使用同一套代码。

- **纯 Go、`CGO_ENABLED=0`**：所有组件在无 cgo 环境下编译与测试通过，可直接交叉编译三平台与 wasm。
- **完整的 MD3E 设计系统**：HCT 色彩引擎、7 种 DynamicScheme Variant、3 档 Contrast、45+ 语义色角色、10 档形状、15 档字体、6 级 Elevation。
- **弹簧动效而非贝塞尔近似**：`motion` 包把 MD3E 的 `(ζ, k)` 精确换算为物理阻尼系数 `c = 2ζ√k`，组件层不手写裸 `ui.SpringOptions`。
- **35 种内置形状 + 双向 Cubic Bézier Morph**：形状可由 `ui.Path` 绘制、裁切与连续变形。
- **按需链接（Zero-Bloat）**：无 `init()` 全局注册表，只使用 `Button` 的应用不会链接进未使用的重型组件或图标数据。
- **完整可测试性**：所有组件可被 `ui.NewTester` 以虚拟时钟驱动，零 `time.Sleep`、零真实窗口依赖。

---

## 环境要求

- **Go 1.27.3**（与 `mygo/go.mod` 对齐）。
- 仓库内的 `mygo/` 是上游框架的本地只读参考与 `replace` 目标（已被 `.gitignore` 忽略），**禁止修改**。
- 根 `go.mod` 使用本地替换：

```text
require github.com/egoist/mygo v0.0.0
replace github.com/egoist/mygo => ./mygo
```

> 如需使用真实上游版本，把 `replace` 指向对应的 `github.com/egoist/mygo` 发布标签即可。

---

## 快速开始

### 运行交互式画廊

`examples/gallery` 是 M1–M8 全部组件的可交互总览：多页面 `ui.Router`、折叠 `NavigationRail`，顶栏实时切换 Seed / Variant / Contrast / Light-Dark / MotionScheme / ReduceMotion / RTL。

```sh
CGO_ENABLED=0 go run ./examples/gallery
```

画廊属于根模块（`mygo-md3e/examples/gallery`），因此根目录的 `go build ./...`、`go test ./...` 与 `go tool mygo vet ./...` 会自动覆盖它。

### 最小 Hello MD3E

```go
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/tokens"
)

type app struct {
	name    string
	notify  bool
}

func (a *app) view(c *ui.Context) {
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	md3e.ThemeScope(c, th, func() {
		cs := th.Colors
		ui.Column(c).Fill().Background(cs.Surface).Padding(32).Gap(16).Children(func() {
			md3e.Text(c, "Hello MD3E", tokens.HeadlineMedium, true)
			md3e.TextField(c, &a.name, md3e.TextFieldOptions{
				Variant: md3e.TextFieldOutlined,
				Label:   "Name",
			})
			md3e.Switch(c, &a.notify, md3e.SwitchOptions{Label: "Notifications"})
			if md3e.Button(c, "Greet", md3e.ButtonOptions{Variant: md3e.ButtonFilled}).Clicked() {
				log.Println("Hello,", a.name)
			}
		})
	}).Fill()
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hello MD3E",
			Width:   560,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

要点：

1. 所有组件首参数是 `*ui.Context`，状态通过指针传入（`*bool`、`*float64`、`*string`…），构造时即写入用户输入。
2. 有状态控件必须在构造前用 `c.Key(...)` 指定 key；容器在读取状态前用 `e.Key(...)`。
3. `md3e.ThemeScope` 同时桥接 `ui.Themed`，因此页面内通过 `ui.Overlay` 打开的 Dialog / Sheet / Menu / Modal Picker / Snackbar 都继承同一主题。
4. `ThemeScope` 返回的元素默认按内容高度布局。作为整窗根包装、且内部使用 `ui.Column(...).Fill()` 或 `ui.Scroll(...).Grow(1)` 时，必须链式调用 `.Fill()`（如上面的 `}).Fill()`），否则 `100%` 高度会解析为无界并撑满整页内容，滚动容器将没有可滚动距离。局部或卡片内的 `ThemeScope` 不要加 `.Fill()`。

---

## 目录结构

```text
.
├── tokens/            # M1：HCT 色彩、DynamicScheme、形状/字体/Elevation 令牌
├── motion/            # M2：Expressive / Standard 弹簧，ζ → c 换算与 12 档预设
├── shape/             # M2：35 种 MD3E 形状与双向 Cubic Bézier Morph
├── internal/mdicon/   # M3：内置零依赖矢量图标
├── theme.go           # M3：Theme / ThemeScope / CurrentTheme / 系统偏好桥接
├── surface.go         # M3：Surface、tonal elevation、阴影、文本辅助
├── ripple.go          # M3：StateLayer 与 Expressive Ripple
├── button.go …        # M4：按钮、分组、SplitButton、FAB、Toolbar、AppBar
├── switch.go …        # M5：选择控件、滑块、Chips、TextField
├── loading.go …       # M6：LoadingIndicator、四类 Progress、Badge、Tooltip、Snackbar
├── card.go …          # M7：Card、List、导航、Tabs、Carousel、Dialog、Sheet、Menu
├── datepicker.go      # M8：DatePicker / DateRangePicker
├── timepicker.go      # M8：TimePicker（Dial / Input，12h / 24h）
├── examples/gallery/  # M8：交互式全量演示（根模块内，无独立 go.mod）
└── examples/m2-demo/  # 早期 M1–M2 独立演示模块（单独 go test）
```

依赖方向严格单向：`tokens → motion → shape`、根包 `md3e` → 上述子包与 `mygo/ui`；子包**不得**导入根包。

---

## 组件总表

### M1 · 设计令牌（`tokens`）

| 能力 | 入口 |
| --- | --- |
| HCT / CAM16 色彩空间 | `tokens.HCT`、`tokens.ContrastRatio` |
| 色调调色板 | `tokens.TonalPalette` |
| 动态配色 | `tokens.NewDynamicScheme`（TonalSpot / Expressive / Vibrant / Neutral / Monochrome / Fidelity / Content） |
| 对比度 | `tokens.ContrastStandard / ContrastMedium / ContrastHigh` |
| 形状与字体 | `tokens.ShapeScale`、`tokens.TypeRole`、`tokens.NewTypographyScale` |
| 高度 | `tokens.ElevationLevel`（Level0–5） |

### M2 · 动效与形状（`motion` / `shape`）

| 能力 | 入口 |
| --- | --- |
| 弹簧预设 | `motion.Expressive`、`motion.Standard` + 12 个 `motion.Token` |
| 规格与动画 | `motion.SpecOf`、`motion.Spring`、`motion.Resolve` |
| 35 种形状 | `shape.All()`、`shape.ByName` |
| 形变 | `shape.NewMorph`、`Morph.AsPath`、`shape.DrawMorph`、`shape.ClipShape` |

### M3 · 主题与基础设施

`Theme`、`ThemeFromSeed`、`ThemeOptions`、`DefaultTheme`、`ThemeScope`、`CurrentTheme`、`Surface`、`SurfaceColorForElevation`、`ApplyTonalElevation`、`ApplyElevationShadows`、`Text`、`ApplyTypeStyle`、`AttachStateLayer`、`AttachRipple`、`IconDraw`。

### M4 · 操作与栏

| 组件 | 入口 |
| --- | --- |
| Button（5 变体 × 5 尺寸 × 2 形状） | `Button` |
| IconButton / ToggleIconButton | `IconButton`、`ToggleIconButton` |
| ButtonGroup（Standard / Connected / Multi / Actions） | `ButtonGroup`、`ButtonGroupMulti`、`ButtonGroupActions` |
| SplitButton | `SplitButton` |
| FAB / ExtendedFAB / FABMenu | `FAB`、`ExtendedFAB`、`FABMenu` |
| FloatingToolbar（水平/垂直，可耦合 FAB） | `FloatingToolbar` |
| TopAppBar（Small / CenterAligned / Medium / Large） | `TopAppBar` |
| SearchAppBar | `SearchAppBar` |

### M5 · 选择与输入

| 组件 | 入口 |
| --- | --- |
| Switch | `Switch` |
| Checkbox（含三态） | `Checkbox`、`CheckboxState` |
| Radio / RadioGroup | `Radio`、`RadioGroup` |
| Slider / RangeSlider（步进、停靠点、Centered） | `Slider`、`RangeSlider` |
| Chips（Assist / Suggestion / Filter / Input） | `AssistChip`、`SuggestionChip`、`FilterChip`、`InputChip`、`ChipGroup` |
| TextField / TextArea | `TextField`、`TextArea` |

### M6 · 反馈与指示器

| 组件 | 入口 |
| --- | --- |
| LoadingIndicator（Standard / Contained） | `LoadingIndicator` |
| Linear / LinearWavy / Circular / CircularWavy | `LinearProgress`、`LinearWavyProgress`、`CircularProgress`、`CircularWavyProgress` |
| Badge / SmallBadge / BadgedBox | `Badge`、`SmallBadge`、`BadgedBox` |
| PlainTooltip / RichTooltip | `PlainTooltip`、`RichTooltip` |
| Snackbar | `SnackbarHost`、`SnackbarState` |

### M7 · 容器、导航与浮层

| 组件 | 入口 |
| --- | --- |
| Card（Elevated / Filled / Outlined / 可点击） | `ElevatedCard`、`FilledCard`、`OutlinedCard`、`Card` |
| Divider | `Divider` |
| Expressive List（分段、槽位） | `List`、`ListItem` |
| NavigationBar | `NavigationBar` |
| NavigationRail（折叠/展开） | `NavigationRail` |
| NavigationDrawer（Modal / Inline） | `NavigationDrawer` |
| PrimaryTabs / SecondaryTabs（可滚动） | `PrimaryTabs`、`SecondaryTabs` |
| Carousel（Hero / MultiBrowse / Uncontained） | `Carousel` |
| AlertDialog / FullScreenDialog | `AlertDialog`、`FullScreenDialog` |
| BottomSheet（Modal / Standard，Detents） | `BottomSheet` |
| Menu（Dropdown / MenuButton / ContextMenu） | `DropdownMenu`、`MenuButton`、`ContextMenu` |

### M8 · 选择器与画廊

| 组件 | 入口 |
| --- | --- |
| DatePicker（Docked / Modal） | `DatePicker` |
| DateRangePicker | `DateRangePicker` |
| TimePicker（Dial / Input，12h / 24h） | `TimePicker` |
| 全量交互画廊 | `examples/gallery` |

---

## 主题、动效与可访问性

### ThemeScope 与动态配色

```go
th := md3e.ThemeFromSeed(ui.Hex("#6750A4"), md3e.ThemeOptions{
	Dark:         true,
	Variant:      tokens.Expressive,
	Contrast:     tokens.ContrastHigh,
	Motion:       motion.Expressive,
	ReduceMotion: false,
})
md3e.ThemeScope(c, th, func() {
	// 页面与它打开的所有浮层共享该主题
})
```

- `ThemeScope` 会把 `*md3e.Theme` 同时桥接为 `ui.Theme`，`CurrentTheme(c)` 在页面、`ui.Overlay` 与嵌套作用域中都能取回同一主题。
- `ThemeFromSeed` 计算完整 HCT 色板，建议在应用状态中缓存，仅当 Seed / Variant / Contrast / Dark 变化时重建（画廊即如此）。

### MotionScheme

```go
spec := motion.SpecOf(motion.Expressive, motion.FastSpatial)
anim := motion.Resolve(motion.Expressive, motion.DefaultSpatial, c.Preferences().ReduceMotion)
```

> **不要手写 `ui.SpringOptions{Damping: ζ}`**：MyGo 的 `Damping` 是物理阻尼系数，不是阻尼比。请始终经由 `motion` 包构造。

### RTL

- 布局使用逻辑边：`PaddingStart/End`、`MarginStart/End`、`Start/End`、`AnchorTopStart` 等。
- 需要整体镜像时可对子树设置 `el.Direction(ui.RightToLeft)`；`ui.Overlay` 会继承调用处的方向，因此 Dialog / Sheet / Menu / Picker 一并镜像。
- 进度、滑块、导航指示器与方向图标读取 `c.Direction()` 处理翻转。

### ReduceMotion

主题或系统偏好开启后，弹簧与形态动画直接收敛到目标态，并且**不再请求持续帧**（`tt.FramePending() == false`）；LoadingIndicator 定格为首个形状，不定量进度条显示静止段。

### 生命周期铁律

- `ui.Element`、`*ui.Context` 与 `ui.*Parts` 只在当前 build pass 内有效，禁止存入 struct、全局变量、跨帧闭包或 goroutine。
- 状态通过指针或 `ui.Local(e, key, init)` 持久化；跨帧身份与焦点用 `ui.Handle`（`.Bind(&h)`）。
- 有状态控件在构造前 `c.Key(k)`；容器在读取状态前 `e.Key(k)`；兄弟节点 key 不得重复。
- 上述规则由 `go tool mygo vet ./...` 与 `ui.NewTester` 强制检查。

---

## 测试

所有 UI 测试使用 `ui.NewTester` 与虚拟时钟，不等待真实时间：

```go
tt := ui.NewTester(app.view, 1280, 800)
tt.Advance(300 * time.Millisecond) // 推进虚拟时钟
if tt.FramePending() {
	// 动画尚未收敛
}
```

覆盖率重点：

- 交互状态转换（点击、拖动、键盘、禁用态拦截）；
- 弹簧收敛（推进后 `FramePending() == false`）与 `ReduceMotion` 行为；
- 无障碍树（`tt.AccessByRole` / `tt.AccessAll`，角色、名称、值、状态、禁用）；
- 自定义绘制组件的像素与几何断言（Shape / Morph / WavyProgress / LoadingIndicator / Ripple / Picker）。

### Golden 基线

`golden_test.go` 用 `ui/golden` 锁定**无文字依赖**的纯几何渲染（35 形状 Morph 中间态、LoadingIndicator、四类进度条、Switch / Slider / RangeSlider 轨道、Expressive 分段列表），以规避各平台字体光栅化差异：

```sh
# 生成/更新基线（仅在有意修改视觉时）
MYGO_UPDATE_GOLDEN=1 CGO_ENABLED=0 go test -count=1 -run TestGolden .

# 正常核对：不带 MYGO_UPDATE_GOLDEN，确保无 *.got.png 残留
CGO_ENABLED=0 go test -count=1 ./...
```

比对失败时会写出 `testdata/<name>.got.png` 供对照；请勿提交 `*.got.png`。

---

## 全套校验与跨平台构建

```sh
# 当前平台：零 cgo 编译、静态分析、生命周期检查、全量测试
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
go tool mygo vet ./...
CGO_ENABLED=0 go test -count=1 ./...

# 画廊专项
CGO_ENABLED=0 go test -count=1 ./examples/gallery/...
CGO_ENABLED=0 go build -o gallery_tmp ./examples/gallery && rm -f gallery_tmp   # Windows: Remove-Item gallery_tmp.exe

# 独立模块示例
cd examples/m2-demo && CGO_ENABLED=0 go test -count=1 ./... && cd ../..

# 跨平台交叉编译
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=js      GOARCH=wasm  go build ./...
```

> Windows 下无 cgo 工具链时不要执行 `go test -race`（依赖 cgo），统一使用 `CGO_ENABLED=0`。

---

## 工程约定

- **不修改 `mygo/`**，不导入 `github.com/egoist/mygo/internal/...`，不引入多余第三方依赖。
- **零 `init()` 注册表**，保持按需链接。
- **一点一提交**：每个 `plan.md` 任务点通过四重门禁后独立提交，保证任一历史提交都可通过 `CGO_ENABLED=0 go test ./...`。
- **不提交产物**：`*.exe`、`*.test`、`*.got.png`、`build/`、`.mygo/`。
