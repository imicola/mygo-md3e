# mygo-md3e

[English](README.md) | 简体中文

**mygo-md3e** 是构建在 [MyGo](https://github.com/egoist/mygo) 原生 GPU UI 工具包（`github.com/egoist/mygo/ui`）之上的 **Material Design 3 Expressive (MD3E)** 原生组件库。纯 Go 实现，零 HTML、零 WebView、零 cgo（`CGO_ENABLED=0`），使用同一套代码同时支持桌面端（macOS / Windows / Linux）与浏览器（`GOOS=js GOARCH=wasm`）。

## 核心特性

- **纯 Go、`CGO_ENABLED=0`**：无 cgo 环境下即可完成编译、测试与运行，并支持 `linux/amd64`、`darwin/arm64`、`js/wasm` 跨平台编译。
- **完整 MD3E 设计令牌（`tokens`）**：HCT / CAM16 色彩科学、`TonalPalette`、7 种 `DynamicScheme` Variant（`TonalSpot`、`Expressive`、`Vibrant`、`Neutral`、`Monochrome`、`Fidelity`、`Content`）、3 档 Contrast、45+ 语义色角色、10 档形状圆角、XS–XL 五档尺寸、30 组 Standard / Emphasized 排版样式及 6 级 Elevation。
- **物理弹簧动效（`motion`）**：12 组 Expressive / Standard 弹簧预设（Spatial 与 Effects），将 MD3E 的阻尼比 $\zeta$ 与刚度 $k$ 精确换算为 MyGo 物理阻尼系数 $c = 2\zeta\sqrt{k}$（$m=1$），Effects 类令牌启用 Oklab 插值与临界阻尼无过冲。
- **35 种表现力形状与双向变形（`shape`）**：内置 `Circle` 至 `Heart` 共 35 种 `RoundedPolygon` 几何基元与双向 Cubic Bézier Morph，可直接通过 `ui.Path` 绘制与裁切。
- **零膨胀矢量图标子包（`symbols`）**：内置 252 个 24×24 圆角矢量符号（分 7 个功能域），零 `init()`、零全局注册表，实现单图标级死代码消除；`symbols.<Name>.Draw` 可直接传入任意 `md3e.IconDraw` 槽位。
- **全套 MD3E 原生组件（`md3e`）**：覆盖按钮/异形形变按钮（`ShapeButton` / `MorphIconButton`）/按钮组/SplitButton/FAB/FABMenu/工具栏/AppBar、选择控件/滑块/Chips/输入框、形状变形加载器/波浪进度条/Badge/Tooltip/Snackbar、卡片/`SwitchCard`/`ActionTile`/分割线/分段列表/导航栏-侧栏-抽屉/下划线与胶囊分段 Tabs、`Tag`/`StatusPill`/`Avatar`/`Skeleton`/`EmptyState`/`Banner`、Carousel/Dialog/BottomSheet/Menu、Date/Time Picker 与桌面 `WindowTitleBar`/Adaptive 布局。
- **按需链接（Zero-Bloat）**：无包级 `init()` 全局注册表，仅使用 `Button` 的应用不会链接进未使用的重型组件或多余内置图标。
- **无障碍、RTL 与无头可测**：构建在 `ui.*Base` 原语之上，完整支持键盘导航、`ReduceMotion` 零空转帧收敛、从右到左（RTL）布局镜像，以及基于 `ui.NewTester` 虚拟时钟和 `ui/golden` 的自动化测试。

---

## 环境要求与安装

- **Go 1.27.3+**（与上游 `github.com/egoist/mygo` 对齐）。
- 直接依赖上游 `github.com/egoist/mygo`（`v0.10.1-0.20261119113012-61ea5567ab14`），克隆后即可直接构建运行，无需本地 `./mygo` 或 `replace` 配置。

在你的 Go 模块中安装：

```sh
go get github.com/imicola/mygo-md3e@latest
```

---

## 快速开始

### 运行交互式组件画廊

`examples/gallery` 是覆盖全部组件与设计令牌的交互式多页面 `ui.Router` 展示应用，带可折叠 `NavigationRail`，支持实时切换 Seed Color、Variant、Contrast、Light/Dark、MotionScheme、`ReduceMotion` 与 RTL：

```sh
CGO_ENABLED=0 go run ./examples/gallery
```

`examples/desktop-demo` 是 PulseDesk：一个紧凑的桌面控制中心示例，把 M9 外壳（`WindowTitleBar`、三档密度、自适应列表详情与辅助面板、Bento 网格）与 M10 表现层（`ShapeButton`/`MorphIconButton`、`PillTabs`/`SegmentedTabs`、Tag、StatusPill、Avatar、Skeleton、EmptyState、Banner）以及零膨胀 `symbols` 图标包组合在一起：

```sh
CGO_ENABLED=0 go run ./examples/desktop-demo
```

此外，`examples/m2-demo` 保留了针对 `tokens` / `motion` / `shape` 的独立预览模块：

```sh
cd examples/m2-demo
CGO_ENABLED=0 go run .
```

### 最小 Hello MD3E 示例

```go
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

type app struct {
	theme  *md3e.Theme
	name   string
	notify bool
}

func (a *app) view(c *ui.Context) {
	if a.theme == nil {
		a.theme = md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	}
	md3e.ThemeScope(c, a.theme, func() {
		cs := a.theme.Colors
		ui.Column(c).Fill().Background(cs.Surface).Padding(32).Gap(16).Children(func() {
			md3e.Text(c, "Hello MD3E", tokens.HeadlineMedium, true)
			md3e.TextField(c.Key("name"), &a.name, md3e.TextFieldOptions{
				Variant: md3e.TextFieldOutlined,
				Label:   "Name",
			})
			md3e.Switch(c.Key("notify"), &a.notify, md3e.SwitchOptions{Label: "Notifications"})
			if md3e.Button(c.Key("greet"), "Greet", md3e.ButtonOptions{Variant: md3e.ButtonFilled}).Clicked() {
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

使用要点：

1. 所有组件首参数均为 `*ui.Context`，状态通过指针传入（`*bool`、`*float64`、`*int`、`*string`、`*time.Time`），构造时即应用用户输入。
2. 有状态控件必须在构造前使用 `c.Key(...)` 指定唯一 key；容器在读取状态或调用 `ui.Local` 前使用 `e.Key(...)`。
3. `md3e.ThemeScope` 会同步桥接 `ui.Themed`，因此内部通过 `ui.Overlay` 弹出的 Dialog / Sheet / Menu / Modal Picker / Snackbar 都会自动继承同一主题。
4. `ThemeScope` 默认按内容高度布局。作为整窗根包装、且内部包含 `.Fill()` 或 `.Grow(1)` 滚动容器时，请在其后链式调用 `.Fill()`（如上所示）；局部或卡片内的 `ThemeScope` 无需加 `.Fill()`。

---

## 目录结构与分层

```text
.
├── tokens/            # HCT/CAM16 色彩引擎、TonalPalette、DynamicScheme、形状/字体/Elevation 令牌
├── motion/            # Expressive / Standard 物理弹簧系统（ζ -> c 换算与 12 档预设）
├── shape/             # RoundedPolygon 基元、35 种内置 MD3E 形状与双向 Cubic Bézier Morph
├── symbols/           # 252 个零膨胀 24×24 圆角矢量符号子包（单图标级 DCE，Draw 兼容 IconDraw）
├── internal/mdicon/   # 内置零全局注册表的 24×24 矢量图标
├── theme.go           # Theme（含 Density 与 MonoFont）/ ThemeFromSeed / DefaultTheme / ThemeScope / CurrentTheme
├── surface.go         # Surface 容器、Tonal Elevation、阴影、Hairline 微高光、渐变与 Backdrop 装饰槽
├── typography.go      # Text / TextWith / TabularNums / TruncateMode / ApplyTypeStyle / TypeStyleOf
├── ripple.go          # AttachStateLayer / AttachRipple（DrawOver 态层与指针起点涟漪）
├── button.go …        # 操作与栏：Button、IconButton、ButtonGroup、SplitButton、FAB、FABMenu、FloatingToolbar、AppBar
├── switch.go …        # 选择与输入：Switch、Checkbox、Radio、Slider、RangeSlider、Chips、TextField、TextArea
├── loading.go …       # 指示器与反馈：LoadingIndicator、Linear/Circular/Wavy 进度条、Badge、Tooltip、Snackbar
├── card.go …          # 容器、导航与浮层：Card、Divider、List、NavBar/Rail/Drawer、Tabs、Carousel、Dialog、Sheet、Menu
├── datepicker.go      # DatePicker（Docked / Modal）与 DateRangePicker
├── timepicker.go      # TimePicker（Dial / Input，12h / 24h）
├── shell.go           # 桌面标题栏 WindowTitleBar 与 WindowController
├── adaptive.go        # WindowSizeClass、AdaptiveScaffold、ListDetailPane、SupportingPane、BentoGrid/BentoSpan
├── shapebutton.go     # ShapeButton、MorphIconButton 与 IconBadge
├── pilltabs.go        # PillTabs 与 SegmentedTabs
├── status.go          # SemanticTone、Tag、StatusPill 与 Avatar
├── tiles.go           # SwitchCard 与 ActionTile
├── feedback_extra.go  # Skeleton、EmptyState 与 Banner
├── docs/              # 架构设计规范与 MD3E 技术规格基线
└── examples/
    ├── gallery/       # 全量交互式展示画廊（属于根模块）
    └── m2-demo/       # M1–M2 令牌/动效/形状独立演示模块
```

包依赖严格保持单向：`tokens` → `motion` → `shape` → 根包 `md3e`，`symbols` 仅依赖 `github.com/egoist/mygo/ui` 与标准库（子包绝不导入 `md3e`）。

---

## 组件与能力总表

### 设计令牌（`tokens`）

| 能力 | 主要入口 |
| --- | --- |
| HCT / CAM16 色彩空间与对比度 | `tokens.HCT`、`tokens.HCTFromColor`、`tokens.CAM16`、`tokens.ContrastRatio`、`tokens.ContrastTone` |
| 色调调色板与动态配色 | `tokens.TonalPalette`、`tokens.NewDynamicScheme`（7 种 Variant：`TonalSpot` / `Expressive` / `Vibrant` / `Neutral` / `Monochrome` / `Fidelity` / `Content`） |
| 对比度等级 | `tokens.ContrastStandard`、`tokens.ContrastMedium`、`tokens.ContrastHigh` |
| 形状、尺寸、密度、字体与高度 | `tokens.ShapeScale`（`ShapeNone`..`ShapeFull`）、`tokens.SizeTier`（`SizeXS`..`SizeXL`）、`tokens.Density`（`DensityComfortable` / `DensityDesktop` / `DensityCompact`）、`tokens.DensitySpec`、`tokens.TypeRole`、`tokens.NewTypographyScale`、`tokens.ElevationLevel`（`Level0`..`Level5`） |

### 动效与形状（`motion` / `shape`）

| 能力 | 主要入口 |
| --- | --- |
| 弹簧方案与令牌 | `motion.Expressive`、`motion.Standard`、`motion.FastSpatial`..`motion.SlowEffects` |
| 规格与动画求解 | `motion.SpecOf`、`motion.Spring`、`motion.Resolve`、`motion.Snap` |
| 35 种内置形状 | `shape.All()`、`shape.ByName`、`shape.Circle`..`shape.Heart`、`shape.RegularPolygon`、`shape.Star` |
| 双向变形与裁切 | `shape.NewMorph`、`Morph.AsPath`、`shape.DrawMorph`、`shape.ClipShape` |

### 主题与基础设施（`md3e`）

`Theme`、`ThemeOptions`、`ThemeFromSeed`、`DefaultTheme`、`ThemeScope`、`CurrentTheme`、`DefaultSeedColor`、`ShapeScale`、`DefaultShapeScale`、`Surface`、`SurfaceOptions`、`SurfaceColorForElevation`、`ApplyTonalElevation`、`ApplyElevationShadows`、`Text`、`TextWith`、`TextOptions`、`TruncateMode`、`TabularNums`、`ApplyTypeStyle`、`TypeStyleOf`、`AttachStateLayer`、`AttachRipple`、`IconDraw`。

### 操作与栏

| 组件 | 主要入口 |
| --- | --- |
| Button（5 变体 × 5 尺寸 × Round/Square） | `Button` |
| Expressive 异形按钮与图标徽章 | `ShapeButton`、`MorphIconButton`、`IconBadge` |
| IconButton / ToggleIconButton | `IconButton`、`ToggleIconButton` |
| ButtonGroup（Standard / Connected / Multi / Actions） | `ButtonGroup`、`ButtonGroupMulti`、`ButtonGroupActions` |
| SplitButton | `SplitButton` |
| FAB / ExtendedFAB / FABMenu（2–6 项） | `FAB`、`ExtendedFAB`、`FABMenu` |
| FloatingToolbar（水平/垂直，可耦合 FAB） | `FloatingToolbar` |
| TopAppBar / SearchAppBar | `TopAppBar`、`SearchAppBar` |

### 选择与输入

| 组件 | 主要入口 |
| --- | --- |
| Switch | `Switch` |
| Checkbox（支持三态） | `Checkbox`、`CheckboxState` |
| Radio / RadioGroup | `Radio`、`RadioGroup` |
| Slider / RangeSlider（步进、停靠点、Centered、水滴指示气泡） | `Slider`、`RangeSlider` |
| Chips | `AssistChip`、`SuggestionChip`、`FilterChip`、`InputChip`、`ChipGroup` |
| TextField / TextArea（Filled / Outlined） | `TextField`、`TextArea` |

### 指示器与反馈

| 组件 | 主要入口 |
| --- | --- |
| 多边形变形加载器（Standard / Contained） | `LoadingIndicator`、`DefaultLoadingPolygons` |
| 线性与环形进度条（标准 / Wavy） | `LinearProgress`、`LinearWavyProgress`、`CircularProgress`、`CircularWavyProgress` |
| Badge | `Badge`、`SmallBadge`、`BadgedBox` |
| 标签、状态胶囊与头像 | `Tag`、`StatusPill`、`Avatar`、`SemanticTone` |
| 骨架屏、空状态与提示横幅 | `Skeleton`、`EmptyState`、`Banner` |
| Tooltip | `PlainTooltip`、`RichTooltip` |
| Snackbar | `SnackbarHost`、`SnackbarState` |

### 容器、导航、浮层与选择器

| 组件 | 主要入口 |
| --- | --- |
| Card 与 Divider | `ElevatedCard`、`FilledCard`、`OutlinedCard`、`Card`、`Divider` |
| 卡片式快捷开关与操作行 | `SwitchCard`、`ActionTile` |
| Expressive 分段列表 | `List`、`ListItem` |
| 导航组件 | `NavigationBar`、`NavigationRail`、`NavigationDrawer` |
| 滑动指示器与胶囊分段 Tabs | `PrimaryTabs`、`SecondaryTabs`、`PillTabs`、`SegmentedTabs` |
| Carousel（Hero / MultiBrowse / Uncontained） | `Carousel` |
| 对话框与底部抽屉 | `AlertDialog`、`FullScreenDialog`、`BottomSheet` |
| 菜单 | `DropdownMenu`、`MenuButton`、`ContextMenu` |
| 日期与时间选择器 | `DatePicker`、`DateRangePicker`、`TimePicker` |
| 桌面外壳与 Adaptive 布局 | `WindowTitleBar`、`WindowController`、`WindowSizeClass`、`WindowSizeClassForWidth`、`WindowSizeClassOf`、`AdaptiveScaffold`、`ListDetailPane`、`SupportingPane`、`BentoGrid`、`BentoSpan` |
| 零膨胀矢量图标子包（`symbols`） | 252 个圆角 24×24 图标（`symbols.Home`、`symbols.PlayArrow`、`symbols.Pause`、`symbols.Settings`、`symbols.Terminal` 等）、`symbols.Custom`、`symbols.CustomEvenOdd` |

### 使用零膨胀 `symbols` 与异形交互组件

```go
import (
	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// 每个图标的 Draw 方法值直接匹配 md3e.IconDraw，且按单图标级 DCE 剪裁：
md3e.MorphIconButton(c.Key("play"), &playing, "Play or pause", symbols.PlayArrow.Draw,
	md3e.MorphIconButtonOptions{
		Size:          tokens.SizeM,
		Shape:         shape.Cookie9Sided(),
		SelectedShape: shape.Circle(),
		SelectedIcon:  symbols.Pause.Draw,
	})

md3e.SwitchCard(c.Key("proxy"), "System Proxy", &proxyOn, md3e.SwitchCardOptions{
	Subtitle:    "自动接管桌面系统代理",
	LeadingIcon: symbols.Public.Draw,
	Hairline:    true,
})
```

---

## 主题、动效、RTL 与无障碍

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
	// 子树及所有 ui.Overlay 浮层共享同一主题
})
```

`ThemeFromSeed` 会计算完整 HCT 色板，建议在应用状态中缓存 `*md3e.Theme` 实例，仅在 Seed / Variant / Contrast / Dark 变化时重建。

### MotionScheme

```go
spec := motion.SpecOf(motion.Expressive, motion.FastSpatial)
anim := motion.Resolve(motion.Expressive, motion.DefaultSpatial, c.Preferences().ReduceMotion)
```

> 请始终通过 `motion` 包构建弹簧动画，不要手写 `ui.SpringOptions{Damping: ζ}`：MyGo 的 `Damping` 是物理阻尼系数 $c$，而非无量纲阻尼比 $\zeta$。

### RTL 与 ReduceMotion

- 布局边距与锚点使用逻辑方法（`PaddingStart`、`PaddingEnd`、`MarginStart`、`MarginEnd`、`AnchorTopStart` 等）。设置 `.Direction(ui.RightToLeft)` 会镜像布局，并自动继承到 `ui.Overlay`（Dialog / Sheet / Menu / Picker）。
- 开启 `ReduceMotion` 后，弹簧动画直接收敛到目标态并停止请求持续动画帧（`tt.FramePending() == false`），`LoadingIndicator` 停留在初始形状，不定量进度条显示静止样式。

---

## 测试与跨平台校验

所有 UI 测试通过 `ui.NewTester` 与虚拟时钟运行，无需真实窗口与 `time.Sleep`：

```go
tt := ui.NewTester(app.view, 1280, 800)
tt.Advance(300 * time.Millisecond)
if tt.FramePending() {
	// 动画尚未收敛
}
```

`golden_test.go` 使用 `github.com/egoist/mygo/ui/golden` 锁定无文字依赖的纯几何渲染基线（`testdata/<name>.png` 及存在时的非 Windows 平台基线 `testdata/<name>_<GOOS>.png`），规避跨平台字体与光栅化差异：

```sh
# 日常校验（不会残留 *.got.png）
CGO_ENABLED=0 go test -count=1 ./...

# 仅在有意调整视觉输出时更新 Golden 基线
MYGO_UPDATE_GOLDEN=1 CGO_ENABLED=0 go test -count=1 -run TestGolden .
```

### 全套构建与交叉编译命令

```sh
# 零 cgo 编译、静态分析、MyGo 生命周期检查（mygo vet 不带 ./... 参数）与全量测试
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
go tool mygo vet
CGO_ENABLED=0 go test -count=1 ./...

# M2 独立示例模块
cd examples/m2-demo && CGO_ENABLED=0 go test -count=1 ./... && cd ../..

# 跨平台交叉编译
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=js      GOARCH=wasm  go build ./...
```

---

## 致谢与许可证

- 基于 [MyGo](https://github.com/egoist/mygo) 构建（MIT 许可证）。
- `internal/mdicon` 内置 24×24 矢量图标路径参考标准 Material Icons 几何定义（Apache-2.0）。
- 本项目遵循 [MIT License](LICENSE) 开源。
