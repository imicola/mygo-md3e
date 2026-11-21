# 架构设计规范（Architecture Guide）

本文档定义了 `mygo-md3e` 的包分层、模块依赖边界、统一组件 API 范式与内部复用基础设施。

---

## 1. 模块与目录布局

Go 模块路径：`github.com/imicola/mygo-md3e`（直接依赖上游 `github.com/egoist/mygo` 与 `tool github.com/egoist/mygo/cmd/mygo`；生命周期静态检查命令为不带路径参数的 `go tool mygo vet`）。

```text
.
├── RULE.md                 # Agent 提交规范与门禁铁律
├── AGENTS.md               # MyGo 技术约束与避坑指南
├── doc.go                  # 根包概览与版本声明
├── docs/
│   ├── architecture.md     # 本文件：架构与 API 设计范式
│   ├── md3e-spec.md        # MD3E 精确设计令牌与组件行为规格
│   └── ROADMAP.md          # Phase 2 演进路线图与里程碑进度
├── go.mod / go.sum
├── tokens/                 # M1/M9: 纯 Go MD3E 设计令牌与 HCT 色彩引擎（无 UI 状态依赖）
│   ├── hct.go              # sRGB/XYZ/Lab/CAM16/HCT 转换、对比度计算
│   ├── palette.go          # TonalPalette & CorePalette
│   ├── scheme.go           # DynamicScheme (TonalSpot/Expressive/Vibrant/...) & 45+ ColorRoles
│   ├── size.go             # XS..XL 尺寸档位（SizeTier / SizeSpec）
│   ├── density.go          # M9: Comfortable / Desktop / Compact 三档密度规格与 ButtonSpec
│   ├── shape.go            # CornerRadius / ShapeScale tokens
│   ├── type.go             # TypographyScale（含 Emphasized 强调样式）
│   └── elevation.go        # Elevation Level 0..5 阴影与表面色调混合参数
├── motion/                 # M2: MD3E Spring 物理动效系统（封装 ui.SpringOptions）
│   └── motion.go           # Expressive / Standard Scheme、Spatial / Effects 12 预设
├── shape/                  # M2: MD3E 35 种几何形状与双射 Cubic Bézier Morph 引擎
│   ├── polygon.go          # RoundedPolygon 构建、CornerRounding、Cubic 序列归一化
│   ├── catalog.go          # 35 种 MD3E 形状定义（Circle, Sunny, Cookie4..12, Heart...）
│   ├── morph.go            # Morph(from, to) 特征对齐插值与 *ui.Path 渲染输出
│   └── draw.go             # ui.Painter 辅助绘制/裁切函数
├── symbols/                # M10: 零膨胀 252 个 24×24 圆角矢量符号子包（独立无环，单图标级 DCE）
│   ├── symbols.go          # Symbol、Custom、CustomEvenOdd、Path、Draw（兼容 md3e.IconDraw）、Stroke
│   ├── navigation.go       # 导航、方向与窗口/布局控制符号（38）
│   ├── actions.go          # 常用操作、CRUD 与筛选符号（44）
│   ├── media.go            # 音视频播放、相机与媒体控制符号（35）
│   ├── editor.go           # 文本编辑、Markdown、表格与排版符号（32）
│   ├── files.go            # 文件、目录、邮件与云存储符号（28）
│   ├── system.go           # 系统、网络、安全、硬件与时间符号（43）
│   └── status.go           # 状态、告警、人员、图表与徽章符号（32）
├── internal/
│   └── mdicon/             # M3: 内置无外部依赖矢量路径/图标（Check, Close, Chevron, Search...）
├── theme.go                # M3/M9: md3e.Theme（含 Density/MonoFont）、ThemeScope、系统偏好桥接、ui.Theme 转换
├── surface.go              # M3/M9: Surface 容器、Elevation 阴影/Tonal 叠色、Hairline/渐变/Backdrop
├── typography.go           # M3/M9: Text、TextWith、TabularNums、ApplyTypeStyle 与 TypeStyleOf 排版辅助
├── ripple.go               # M3: StateLayer 状态层（Hover/Focus/Pressed/Dragged）与 Expressive Ripple
├── button.go               # M4: Expressive Button (Filled/Tonal/Elevated/Outlined/Text × XS-XL)
├── iconbutton.go           # M4: IconButton & ToggleIconButton（Round/Square 变形）
├── buttongroup.go          # M4: ButtonGroup (Connected / Standard 邻居挤压与动态圆角)
├── splitbutton.go          # M4: SplitButton（主操作 + 变形下拉箭头）
├── fab.go                  # M4: FAB (Small/Medium/Large) & ExtendedFAB
├── fabmenu.go              # M4: FABMenu（2-6 项展开菜单）
├── toolbar.go              # M4: FloatingToolbar (水平/垂直 + 耦合 FAB)
├── appbar.go               # M4: TopAppBar (Small/Medium/Large/Centered) & SearchAppBar
├── switch.go               # M5: Expressive Switch
├── checkbox.go             # M5: Checkbox（三态支持）
├── radio.go                # M5: Radio & RadioGroup
├── slider.go               # M5: Slider & RangeSlider（停靠点、水滴气泡、Centered/Discrete）
├── chip.go                 # M5: Assist / Filter / Input / Suggestion Chip
├── textfield.go            # M5: TextField & TextArea（Filled / Outlined + 浮动标签）
├── loading.go              # M6: LoadingIndicator（连续多边形 Morph 加载器）
├── progress.go             # M6: Linear/Circular & LinearWavy/CircularWavy ProgressIndicator
├── badge.go                # M6: Badge & BadgedBox
├── tooltip.go              # M6: PlainTooltip & RichTooltip
├── snackbar.go             # M6: Snackbar & SnackbarHost
├── card.go                 # M7: Elevated / Filled / Outlined Card
├── divider.go              # M7: Horizontal / Vertical Divider
├── list.go                 # M7: Expressive List & ListItem（分段圆角、前/尾槽位）
├── navbar.go               # M7: NavigationBar
├── navrail.go              # M7: NavigationRail（支持折叠/展开、FastSpatial 纵向滑移指示器与可选引导竖条）
├── transition.go           # 页面切换过渡容器（PageTransition / AnimatedContent：FadeThrough / SharedAxisY / SharedAxisX / None）
├── navdrawer.go            # M7: ModalNavigationDrawer & DismissibleNavigationDrawer
├── tabs.go                 # M7: PrimaryTabs & SecondaryTabs
├── carousel.go             # M7: Carousel（Hero / MultiBrowse / Uncontained）
├── dialog.go               # M7: AlertDialog & FullScreenDialog
├── sheet.go                # M7: ModalBottomSheet & StandardBottomSheet
├── menu.go                 # M7: Expressive Menu / DropdownMenu / ContextMenu 样式
├── datepicker.go           # M8: DatePicker (Docked / Modal) & DateRangePicker
├── timepicker.go           # M8: TimePicker (Dial / Input)
├── shell.go                # M9: WindowTitleBar 与 WindowController（零后端耦合的桌面标题栏外壳）
├── adaptive.go             # M9: WindowSizeClass、ContentSizeClassOf、AdaptiveScaffold、ListDetailPane、SupportingPane、BentoGrid
├── shapebutton.go          # M10: ShapeButton、MorphIconButton（双向 Morph + 图标交叉过渡）与 IconBadge
├── pilltabs.go             # M10: PillTabs 与 SegmentedTabs（胶囊滑动指示器 + 徽章）
├── status.go               # M10: SemanticTone、Tag、StatusPill（离屏零帧呼吸点）与 Avatar（异形/图片/状态点）
├── tiles.go                # M10: SwitchCard（Key 隔离防双切）与 ActionTile（RTL 自适应箭头）
├── feedback_extra.go       # M10: Skeleton（Tonal 柔和脉冲骨架屏）、EmptyState 与 Banner
├── golden_test.go          # 纯几何 Golden 视觉基线测试（testdata/）
└── examples/
    ├── m2-demo/            # M2: 独立可运行的令牌/动效/形状预览模块
    └── gallery/            # M8: 交互式 MD3E 全量演示与主题/动效/形状游乐场
```

---

## 2. 依赖方向与解耦规则

1. **严格单向依赖**：
   - `tokens` 仅依赖 Go 标准库 + `github.com/egoist/mygo/ui`（仅使用 `ui.Color` 类型或与 `ui.RGB`/`RGBA` 互转，不依赖布局/引擎）。
   - `motion` 仅依赖 `tokens` 与 `github.com/egoist/mygo/ui`（`Animation`、`SpringOptions`、`Preferences`）。
   - `shape` 仅依赖 `github.com/egoist/mygo/ui`（`Path`、`Painter`、`Rect`、`Point`）。
   - `symbols` 仅依赖 Go 标准库与 `github.com/egoist/mygo/ui`（`Path`、`Painter`、`Rect`、`Color`），不导入 `md3e`，其 `(*Symbol).Draw` 方法值签名与 `md3e.IconDraw` 完全一致。
   - `md3e`（根包）整合 `tokens`、`motion`、`shape`、`internal/mdicon` 与 `github.com/egoist/mygo/ui`。
   - **禁止循环依赖**：`tokens`、`motion`、`shape`、`symbols` 不得导入根包 `md3e`。
2. **零 `init()` 注册表**：任何包不得在 `init()` 中构造大对象或注册全量组件表，确保用户应用只链接实际调用的组件（遵循 MyGo“按需付费”原则）。

---

## 3. 主题与上下文传递机制（Theme Scope）

MyGo 的 `ui.Context` 本身携带有 `*ui.Theme`（通过 `c.Theme()` 读取、`ui.Themed(c, t)` 局部切换）。`md3e` 在此之上提供更丰富的 `md3e.Theme`（包含完整 MD3E `ColorScheme`、`Typography`、`Shape`、`MotionScheme`），桥接原则如下：

1. **默认回退**：若外层未显式设置 `md3e.Theme`，组件内部调用 `md3e.CurrentTheme(c)` 时，应基于 `c.Theme()` 与 `c.Preferences()`（暗色模式 `Dark`、系统强调色 `Accent`、高对比度 `HighContrast`、减少动效 `ReduceMotion`）自动派生默认 MD3E 主题（带轻量化缓存，避免每帧重复计算整套 HCT 色板）。
2. **局部与全局主题作用域**：
   - `md3e.ThemeScope(c *ui.Context, theme *md3e.Theme, fn func()) ui.Element`：内部同时调用 `ui.Themed(c, theme.UITheme())` 并通过元素/上下文作用域传递 `*md3e.Theme`，使得根窗口、`ui.Overlay` 弹出层及嵌套预览区都能正确获取局部主题。
   - 当存在嵌套 `ui.Themed` 且与外层 `md3e.Theme` 不一致时，主题解析应能正确感知暗色/强调色变化，不出现“外层切暗色、内层控件颜色错乱”的问题。

---

## 4. 统一组件 API 设计范式

参照 `mygo/ui` 与 `mygo/ui/mobile` 的 Go-first 惯例：

1. **函数签名规范**：
   - 首参数恒为 `c *ui.Context`。
   - 数据绑定采用指针传递（如 `checked *bool`、`selected *int`、`value *float64`、`text *string`、`open *bool`、`value *time.Time`），遵循 MyGo“构造即写入输入、立即读出变更或通过 `.Changed()` / `OnChanged` 回调观测”的契约。
   - 绝大多数组件返回 root `ui.Element`（`SplitButton` 返回 `(group, action, menuBtn, panel ui.Element)`），以便调用方继续链式调用 `.Width(...)`、`.Disabled(...)`、`.Label(...)`、`.Bind(&handle)` 等。
2. **可变配置（Options Pattern）**：
   - 高频简洁组件提供清晰参数或可选尾参 `opts ...ButtonOptions`（多于 1 个 Options 时显式 `panic`，如 `mygo/plugins/loading` 规范），零值结构体即规范默认。
   - **禁止使用布尔参数堆砌**：用枚举类型（如 `ButtonVariant`、`SizeTier`、`ButtonShape`、`TextFieldVariant`）替代多个 `bool` 标志。
3. **Key 处理警示**：
   - 组合组件内部创建子元素或使用 `ui.Local` / `ui.*Base` 时，不得覆盖外层传入的 `c.Key(...)` 语义！
   - 正确做法：外层通过 `c.Key(k)` 传递给组合组件最外层根元素；内部每个固定子槽位使用自身稳定位置或子级明确 key（如 `c.Key("arrow")`），绝不在外层未消费 key 时抢先创建辅助兄弟节点偷走 `c.Key(...)`。
4. **通用视觉基础设施复用**：
   - 所有容器背景、圆角、阴影及 tonal elevation 通过 `surface.go` 的统一原语绘制，禁止在各组件中硬编码阴影参数或散落写死色值。
   - 所有交互组件的 Hover / Focus / Pressed 状态层叠加及 Ripple 波纹通过 `ripple.go` 的统一附加器（`AttachStateLayer` / `AttachRipple`）处理。
   - 所有内置小图标从 `internal/mdicon` 统一提取，避免多个组件各自复制同一 SVG 路径字符串。
