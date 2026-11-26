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
├── tokens/                 # M1/M9/M12: 纯 Go MD3E 设计令牌、HCT 色彩与封面取色引擎（无 UI 状态依赖）
│   ├── hct.go              # sRGB/XYZ/Lab/CAM16/HCT 转换、对比度计算
│   ├── palette.go          # TonalPalette & CorePalette
│   ├── scheme.go           # DynamicScheme (TonalSpot/Expressive/Vibrant/...) & 45+ ColorRoles
│   ├── extract.go          # M12: QuantizerCelebi（Wu+WSMeans）与 Material Score 动态封面取色（上限 16,384 样本）
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
├── virtuallist.go          # M11: VirtualList（基于 ui.List 的万级虚拟化列表与 DrawOver 态层/涟漪）
├── datatable.go            # M11: DataTable、DataTableCellText 与 ApplyDataTableMenuActions（基于 ui.Table）
├── treeview.go             # M11: TreeView（基于 ui.TreeBase 的缩进引导线、弹簧旋转折叠箭头与本地展开态回退）
├── breadcrumb.go           # M11: Breadcrumb（响应式路径面包屑与 MoreHoriz 溢出下拉菜单）
├── pagination.go           # M11: Pagination（1-based 胶囊分页条、TabularNums 与 MaxButtons 槽位上限）
├── image.go                # M12: Image（根包局部/SVG 图像，形状裁切、Fade-In、Skeleton 与 Error/Retry）
├── media/                  # M12: 业务扩展子包（按需引入，零侵入根包）
│   ├── helpers.go          # media 包内动画、数值防御与文本回退辅助函数
│   ├── netimage.go         # NetImage 与缓存控制（封装 github.com/egoist/mygo/ui/netimage）
│   ├── lightbox.go         # ImageLightbox 全屏模态预览（缩放、平移、RTL 导航、Escape/Scrim）
│   ├── scrubber.go         # 内部共享 Expressive mediaScrubber（波浪/胶囊/缓冲段/时间气泡）
│   ├── player.go           # VideoPlayer（封装 ui.Video / ui/video）与 OpenVideo* 构造器
│   ├── audio.go            # AudioPlayerCard 与 MiniPlayer（memoized 动态取色 + AnimatedThemeScope）
│   ├── image_test.go       # NetImage & ImageLightbox 无头/Mock HTTP/RTL 测试
│   ├── player_test.go      # VideoPlayer（GIF 解码）/ AudioPlayerCard / MiniPlayer 测试
│   └── dce_test.go         # Zero-Bloat / 死代码消除链接边界测试
├── charts/                 # M14: 实时与统计图表扩展子包（按需引入，零侵入根包）
│   ├── doc.go              # charts 子包概览与零膨胀约束
│   ├── helpers.go          # 非有限值防护、视口相交、弹簧入场与零分配圆角路径构造
│   ├── palette.go          # SeriesColor / SeriesPalette（HCT 谐调多序列色板与对比度保障）
│   ├── ringbuffer.go       # RingBuffer（零分配定长实时采样环形缓冲区）
│   ├── sparkline.go        # AreaSparkline（Steffen 平滑曲线、三段渐变、柔光与零分配 Path 复用）
│   ├── stacked.go          # StackedProgress（多段占比条、RTL 镜像、悬停浮标与 TabularNums 图例）
│   ├── metriccard.go       # MetricCard（仪表盘指标卡片、径向氛围底光与圆角贴合 Sparkline/Progress）
│   ├── tooltip.go          # 共享 MD3E RichTooltip 浮动卡片、交互聚焦图例与坐标轴标签
│   ├── linechart.go        # LineChart（封装 ui/chart.Plot/NewLinear/Select）
│   ├── barchart.go         # BarChart（封装 ui/chart.NewBand/Stack/Rect/Select）
│   ├── donutchart.go       # DonutChart（封装 ui/chart.Pie/Sector/SliceAt/Polar）
│   ├── sparkline_test.go   # RingBuffer / SeriesPalette / AreaSparkline 零分配与像素测试
│   ├── dashboard_test.go   # StackedProgress & MetricCard 测试
│   ├── charts_test.go      # LineChart / BarChart / DonutChart 交互、RTL 与弹簧收敛测试
│   └── dce_test.go         # Zero-Bloat / 死代码消除链接边界测试
├── codeview/               # M13.3: ui/editor 主题、字体与滚动条薄桥接（按需引入）
│   ├── doc.go              # codeview 子包概览
│   ├── theme.go            # ThemeFromMD3E / FontFromMD3E / ScrollbarFromMD3E / ApplyToEditor / ApplyToDiff / ConfigureEditor
│   ├── theme_test.go       # Light/Dark/HighContrast 对比度与渲染测试
│   └── dce_test.go         # Zero-Bloat / 死代码消除链接边界测试（仅 ThemeFromMD3E 不拉入 editor.View/DiffView）
├── docview/                # M13: 文档、Markdown、流式渲染与大纲目录子包（按需引入）
│   ├── doc.go              # docview 子包概览
│   ├── helpers.go          # 主题颜色选取与非有限浮点防护工具
│   ├── codeblock.go        # CodeBlock 代码卡片（复制 2s 反馈 / Snackbar / 语法高亮）
│   ├── callout.go          # Callout 五类 MD3E 语义提示块与 CalloutKindFromAlert
│   ├── markdown.go         # MarkdownTheme / MarkdownThemeFor / Markdown / MarkdownStream / Revealed
│   ├── state.go            # State / OutlineItem / ExtractOutline（标题测量与 Scroll/Anchor 跳转）
│   ├── outline.go          # Outline 文档大纲导航目录（配合 SupportingPane）
│   ├── blocks_test.go      # CodeBlock / Callout 交互、退场幽灵、RTL 与 Snackbar 测试
│   ├── markdown_test.go    # Markdown 排版密度、块集成、Stream、Copy as Markdown 与 ReduceMotion 测试
│   ├── outline_test.go     # Outline 滚动联动、Anchor 回退、键盘与 RTL 测试
│   └── dce_test.go         # Zero-Bloat / 死代码消除链接边界测试
├── golden_test.go          # 纯几何 Golden 视觉基线测试（testdata/）
└── examples/
    ├── m2-demo/            # M2: 独立可运行的令牌/动效/形状预览模块
    ├── desktop-demo/       # M9–M10: PulseDesk 紧凑桌面工作站示例
    ├── workbench/          # M15.3: 多场景旗舰工作台（M15.3.a 实时监控仪表盘 + M15.3.b 媒体与图库中心 + M15.3.c 文档与知识工作台）
    └── gallery/            # M8: 交互式 MD3E 全量演示与主题/动效/形状游乐场
```

---

## 2. 依赖方向与解耦规则

1. **严格单向依赖**：
   - `tokens` 仅依赖 Go 标准库 + `github.com/egoist/mygo/ui`（仅使用 `ui.Color` 类型或与 `ui.RGB`/`RGBA` 互转，不依赖布局/引擎）。
   - `motion` 仅依赖 `tokens` 与 `github.com/egoist/mygo/ui`（`Animation`、`SpringOptions`、`Preferences`）。
   - `shape` 仅依赖 `github.com/egoist/mygo/ui`（`Path`、`Painter`、`Rect`、`Point`）。
   - `symbols` 仅依赖 Go 标准库与 `github.com/egoist/mygo/ui`（`Path`、`Painter`、`Rect`、`Color`），不导入 `md3e`，其 `(*Symbol).Draw` 方法值签名与 `md3e.IconDraw` 完全一致。
   - `md3e`（根包）整合 `tokens`、`motion`、`shape`、`internal/mdicon` 与 `github.com/egoist/mygo/ui`；**严禁导入** `media`、`charts`、`codeview`、`docview`、`ui/netimage`、`ui/video`、`ui/chart`、`ui/markdown` 或 `ui/editor`。
   - `media`（扩展媒体子包）单向构建于 `md3e`、`tokens`、`motion`、`shape`、`symbols`、`github.com/egoist/mygo/ui`、`ui/netimage` 与 `ui/video` 之上，各功能块无包级 `init()`，确保仅使用 `NetImage` 时不链接 `ui/video`，仅使用根包 `md3e.Button`/`md3e.Image` 时零引入 `media/`。
   - `charts`（扩展图表子包）单向构建于 `md3e`、`tokens`、`motion`、`github.com/egoist/mygo/ui` 与 `github.com/egoist/mygo/ui/chart` 之上，不导入 `symbols/` 或 `media/`，零 `init()` 注册表，确保仅使用根包 `md3e.Button` 时零引入 `charts/` 与 `ui/chart`，且仅使用 `AreaSparkline`/`RingBuffer` 时不链接 `BarChart`/`DonutChart`。
   - `codeview`（代码编辑器主题桥接）单向构建于 `md3e`、`tokens`、`github.com/egoist/mygo/ui` 与 `github.com/egoist/mygo/ui/editor` 之上，零包级 `init()`；仅调用 `codeview.ThemeFromMD3E` 时不链接 `editor.View`/`editor.DiffView` 重渲染路径。
   - `docview`（文档与 Markdown 子包）单向构建于 `md3e`、`tokens`、`motion`、`shape`、`codeview`、`github.com/egoist/mygo/ui` 与 `ui/markdown` 之上，不导入 `symbols/`、`media/` 或 `charts/`，零包级 `init()`；`Outline`/`State` 路径不拉入 `editor.View`/`editor.DiffView`，且仅使用根包 `md3e.Button` 时零引入 `docview/`、`codeview/` 与 `ui/markdown`。
   - **禁止循环依赖**：`tokens`、`motion`、`shape`、`symbols` 不得导入根包 `md3e`、`media`、`charts`、`codeview` 或 `docview`；`md3e` 不得导入任何业务扩展子包（如 `media`、`charts`、`codeview`、`docview`）。
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
   - `RoundedPolygon.WriteTo(dst, rect)` 将单位方形内的轮廓写入调用方持有的 `ui.Path`。用 `shape.ClipShapeTo` 可以复用同一个缓冲完成裁切。控件将跨帧使用的绘制缓冲放在 `ui.Local` 中。MyGo 在 `FillPath`、`StrokePath` 和 `ClipPath` 调用期间展开路径，回调可以依次复用缓冲。退场元素仍使用其保存的几何和视觉参数绘制。
