# mygo-md3e 第二阶段演进路线图（Phase 2 Roadmap）

> **定位与愿景**：第一阶段（M1–M8）已完成纯 Go Material Design 3 Expressive（MD3E）规范令牌、物理弹簧、35 形状 Morph 与基础组件生态。第二阶段的目标是将 `mygo-md3e` 从 **“符合规范的组件陈列库”** 升级为 **“覆盖 80%+ 现代前端与桌面/跨端开发场景、视觉质感媲美甚至超越 Flutter、但常驻内存与二进制体积压至其 1/5 以下的生产级原生 UI 组件生态”**。

---

## 1. 为什么覆盖 80% 场景但工作量可控？（核心杠杆策略）

直接对标 Jetpack Compose Material 3（1.5+ / M3 Adaptive）与 Flutter Material 3 / `m3e_collection` 时，盲目重造所有复杂引擎会导致项目失控。第二阶段能够**用 30% 的代码量撬动 80%+ 的前端场景**，依赖两大架构杠杆：

### 杠杆 A：充分复用上游 `github.com/egoist/mygo` 已内置的重型底层引擎
经源码核查，当前依赖的上游 `mygo` 已原生提供以下生产级能力，`mygo-md3e` **严禁重造轮子**，只做轻薄的 MD3E 主题桥接、Expressive 形状/动效壳子与交互补全：

| 业务域 | 上游 `mygo` 已有底层能力 | `mygo-md3e` 第二阶段职责（薄封装） |
| :--- | :--- | :--- |
| **桌面窗口外壳** | `mygo.TitleBarHidden` / `TitleBarHiddenInset` + `ui.Element.DragWindow()` | MD3E `WindowTitleBar`、`AppShell`、平台窗口控制按钮与窄侧栏 |
| **自适应多窗格** | `ui.Split` / `ui.SplitPercent`、`ui.Grid`、`ui.SidebarSection` | MD3E Adaptive 断点（Compact / Medium / Expanded）、`ListDetailPane`、`SupportingPane`、`BentoGrid` |
| **虚拟化大数据** | `ui.List`（百万行 ~14.5 µs/帧）、`ui.Table`、`ui.Tree` | MD3E 主题化 `VirtualList`、`DataTable`（排序/列宽/行态层）、`TreeView` |
| **图片与网络图** | `ui.Image`（位图/SVG）+ `ui/netimage`（异步加载/内存磁盘缓存/视口优先） | MD3E `Image` / `NetImage`（Expressive 形状裁切、Tonal 骨架占位、错误回退、封面提取 Dynamic Color 种子、灯箱预览） |
| **音视频播放器** | `ui.Video` + `ui/video`（硬件解码 AVFoundation / Media Foundation / GStreamer / `<video>`）+ GIF/APNG | MD3E `MediaPlayer` / `AudioPlayer` 控件壳（波浪进度条 Scrubbing、异形 Play/Pause Morph 按钮、悬浮控制栏、封面取色） |
| **文档与富文本** | `ui/markdown`（增量 CommonMark、可选中复制、流式支持）、`ui/richtext` | MD3E `MarkdownView` 主题映射（排版尺度、Surface 代码块+一键复制、Callout 提示块、目录 TOC） |
| **实时与统计图表** | `ui/chart`（无样式坐标轴、比例尺、`LineInto` 零分配路径、面积/柱状/饼图） | MD3E `AreaSparkline`、`StackedProgress`、`LineChart`、`BarChart`、`MetricCard` |
| **代码与 Diff** | `ui/editor`（语法高亮、代码编辑、Diff 视图） | 可选极薄主题桥接（将 MD3E `ColorScheme` 映射为 `editor` 配色方案） |
| **数据排版细节** | `FontFeatures("tnum")`、`MaxLines`、`TruncateMiddle()` / `TruncateStart()` | `Theme.MonoFont`、等宽数字辅助函数、`TextOptions` 截断与状态色角色 |

### 杠杆 B：按需子包隔离（严格遵守 Zero-Bloat / DCE 铁律）
重域封装**绝不写入根包 `md3e`**，防止简单工具应用被迫链接 HTTP、Markdown 或视频后端。依赖图保持严格无环：

```text
tokens  ──►  motion  ──►  shape  ──►  md3e (根包：核心美学、布局、表单、虚拟表格/树、基础静态图)
                                        │
        ┌───────────────┬───────────────┼───────────────┬───────────────┐
        ▼               ▼               ▼               ▼               ▼
    symbols/         charts/         media/         docview/       (可选) codeview/
  (零膨胀矢量图标)  (封装 ui/chart) (封装 netimage  (封装 markdown) (薄主题桥接 ui/editor)
                                    & ui/video)
```

---

## 2. 行业对标与 5 大高频应用覆盖矩阵

对标 **Jetpack Compose M3 Expressive + M3 Adaptive**、**Flutter M3 / `m3e_collection`** 以及现代桌面/跨端产品形态，第二阶段瞄准覆盖以下 **5 类占日常开发 80%+ 的核心场景**：

| 应用形态 | 典型代表 | 第二阶段补齐的关键能力 |
| :--- | :--- | :--- |
| **1. 桌面工具与监控仪表盘** | 代理客户端（FlClash 等）、系统监控、DevOps 面板 | `WindowTitleBar`、`Density(Desktop)`、`BentoGrid`、`AreaSparkline`、`StackedProgress`、`PillTabs`、`Tag`/`StatusPill`、`SwitchCard`、`DataTable` |
| **2. 媒体、图库与音视频应用** | 音乐/播客播放器、视频客户端、相册与素材管理 | `Image`（形状裁切/骨架/灯箱）、`media.NetImage`、`media.Player`、`tokens.SeedFromImage`（封面动态取色）、`SupportingPane`（歌词/队列侧栏） |
| **3. 文档、知识库与 AI 对话** | Markdown 笔记、文档阅读器、AI Chat 客户端 | `docview.Markdown`（流式友好、MD3E 排版与代码块复制）、`TreeView`（目录树）、`Breadcrumb`、`ListDetailPane`、平滑流式滚动与骨架屏 |
| **4. 中后台管理与数据 CRUD** | 配置管理、数据库工具、日志检索、任务系统 | `DataTable`（万行虚拟化/排序/列宽/多选）、`VirtualList`、`Pagination`、`EmptyState`、`Banner`、`Stepper` / 表单分组 |
| **5. 移动端/响应式跨端消费应用** | 个人效率 App、信息流、设置与偏好中心 | Adaptive 断点（`NavigationBar` ↔ `NavigationRail` 自动切换）、`PullToRefresh`（Expressive 形变刷新）、`Avatar`、`ShapeButton` |

---

## 3. 第二阶段量化目标（美学 × 性能 × 占用）

1. **美学与交互完成度**：
   - 支持 **桌面紧凑（`Desktop`）/ 标准（`Comfortable`）/ 极简（`Compact`）** 三档密度，彻底摆脱桌面端“手机放大感”。
   - 形成完整的 **3 层深/浅色 Tonal Surface 层次** + 微高光描边 + Hero 有机装饰层 + 35 形状 Morph 按钮体验。
   - 提供开箱即用的 200+ 零膨胀 Material Symbols Rounded 图标，开发体验追平 Flutter `Icons.*`。
2. **性能与资源硬指标**（对标 Flutter 同类桌面仪表盘常驻 UI 内存 ~170 MB）：
   - **稳态帧时间**：复杂多卡片仪表盘单帧 `< 0.5 ms`；100,000 行 `DataTable` 滚动与单行刷新 `< 0.3 ms/帧`。
   - **热路径分配**：`Shape`/`Morph`/`ClipShape` 与 `AreaSparkline` 实时绘制达成 **`0 allocs/op`**。
   - **静止与离屏功耗**：窗口不可见、元素滚动出 `p.Visible()` 或静止态下 **`tt.FramePending() == false`**（0% 空闲 CPU）。
   - **内存占用**：复杂仪表盘 Go Retained Heap `< 5 MB`；完整桌面窗口 RSS 目标控制在 **20–40 MB** 级别（Flutter 的 **1/5 以下**）。
   - **按需链接体积（DCE Guard）**：仅引用 `md3e.Button` 的最小程序不链接 `symbols/`、`charts/`、`media/`、`docview/` 与 35 形状全表。

---

## 4. 分阶段里程碑规划（M9 – M15）

遵循 [RULE.md](../RULE.md) 的**单主题小步提交（One Concern, One Commit）**原则，按优先级划分为 **核心必做（P0）**、**广谱生态子包（P1）** 与 **性能/标杆闭环（P2）**，每个里程碑独立可用、可随时暂停交付。

---

### P0：通用美学、外壳与基础缺口（所有应用都受益）

#### M9 — 桌面美学地基、密度系统与 Adaptive 布局外壳
> **目标**：把视觉质感从“规范可用”拉升到“精致原生桌面/响应式应用级”，为后续所有场景提供统一容器与比例基准。

- [x] **M9.1 密度系统与主题增强（`tokens` + `theme.go` + `typography.go`）**
  - 新增 `tokens.Density`（`DensityComfortable`、`DensityDesktop`、`DensityCompact`），接入 `md3e.ThemeOptions`。
  - `Theme` 新增 `MonoFont` 字段；`typography.go` 新增 `TabularNums`（封装 `FontFeatures("tnum")`）、`TextOptions`（`MaxLines`、`TruncateEnd`、`TruncateMiddle`、`Mono`、`Color`）。
  - 为 `NavRail`、`List`、`Tabs`、`Card`、`Button`、`TextField` 增加合理的 `Density` 缩放与可配置宽度（如桌面侧栏默认 `180–200` DIP 而非僵硬的 `256` DIP）。
- [x] **M9.2 表面层次与卡片质感升级（`surface.go` + `card.go`）**
  - `SurfaceOptions` / `CardOptions` 增加：细腻微高光描边（`Hairline` / 低透明 `OutlineVariant`）、可选渐变底色（`LinearGradient`/`RadialGradient`）、可选背景装饰槽位（`Backdrop func(p *ui.Painter, r ui.Rect)`，用于 Hero 卡片有机波纹/裁剪图案）。
  - 完善 `Card` 圆角档位支持（如 Bento 大圆角 `ShapeLargeIncreased` / `ShapeExtraLarge`）。
- [x] **M9.3 桌面窗口外壳与 Adaptive 多窗格（`shell.go` + `adaptive.go`）**
  - `WindowTitleBar`：基于上游 `TitleBarHidden` + `DragWindow()`，提供无缝融入 `SurfaceContainerLowest` 的跨平台标题栏（内置 Pin/Min/Max/Close 槽位、双击最大化、macOS 左侧交通灯安全留白）。
  - `WindowSizeClass`（`Compact < 600` / `Medium 600..840` / `Expanded >= 840`）与 `AdaptiveScaffold`（自动在 `NavigationBar` ↔ 折叠/展开 `NavigationRail` 间切换）。
  - `ListDetailPane` & `SupportingPane`（基于上游 `ui.Split` 封装带 MD3E 拖拽手柄的自适应双栏/三栏骨架）。
  - `BentoGrid`（基于上游 `ui.Grid` 封装列跨度与断点自排）。

#### M10 — Expressive 异形交互、高频微组件与零膨胀图标库
> **目标**：补齐 Flutter/Compose 生态和现代桌面应用中最常用的“黏合剂组件”与开箱即用的图标体验。

- [ ] **M10.1 异形交互控件（`shapebutton.go`）**
  - `ShapeButton` / `MorphIconButton`：支持传入任意 `shape.RoundedPolygon`（如 `Cookie9Sided`、`SoftBurst`、`Sunny`、`Circle`）并在 Pressed/Selected/Toggled 状态间平滑 Morph（例如播放↔暂停键）。
  - `IconBadge`：带 MD3E Tonal / 异形容器底色的图标徽章（适用于快捷开关行、列表前缀、状态角标）。
- [ ] **M10.2 高频组合与反馈微组件**
  - `PillTabs` / `SegmentedTabs`：胶囊底槽 + 弹簧滑动选中胶囊块 + 可选计数徽章（区别于下划线 `PrimaryTabs`）。
  - `Tag` & `StatusPill`：紧凑只读信息标签与带呼吸/静态圆点的状态胶囊（如 `● 运行中`、`机房`）。
  - `SwitchCard` / `ActionTile`：一行卡片式快捷开关/跳转行（图标徽章 + 标题/副标 + Switch/Chevron）。
  - `Avatar`（文字/图片/异形裁切 + 在线状态点）、`Skeleton`（Tonal 柔和脉冲骨架屏，尊重 `ReduceMotion`）、`EmptyState`（插图/图标 + 标题 + 操作按钮占位）、`Banner`（顶部非模态提示横幅）。
- [ ] **M10.3 零膨胀公开图标子包（`symbols/`）**
  - 导出 200+ 高频 Material Symbols Rounded 24×24 矢量图标（覆盖导航、媒体播放、文档编辑、网络/系统、文件、状态、操作）。
  - 保持零 `init()`、零全局 map、延迟解析与 `ui.Path` 复用，实现单图标级 DCE；提供 `symbols.Custom(svgPath)` 供业务方扩展。

#### M11 — 大数据虚拟化与层级导航组件（根包补齐）
> **目标**：补齐中后台、知识库、连接监控等数据密集场景的通用核心控件。

- [ ] **M11.1 `VirtualList`（`virtuallist.go`）**
  - 基于上游 `ui.List` 封装，支持万级/百万级条目流畅滚动、MD3E 分节圆角/行悬停（`Draw` 态层）、键盘导航与选中态。
- [ ] **M11.2 `DataTable`（`datatable.go`）**
  - 基于上游 `ui.Table` 封装 MD3E 数据表格：吸顶表头、列排序指示器、列宽拖拽、等宽数字右对齐、紧凑/舒适密度、行选中与右键菜单集成。
- [ ] **M11.3 层级导航与分页**
  - `TreeView`（基于上游 `ui.Tree` 封装 MD3E 缩进线、弹簧旋转展开箭头、选中胶囊高亮，适用于文件树/文档目录/配置树）。
  - `Breadcrumb`（路径面包屑，窄屏自动折叠省略号菜单）。
  - `Pagination`（基于 `ButtonGroup` / `IconButton` 组合的分页条）。

---

### P1：广谱业务域扩展子包（按需引入，零侵入根包）

#### M12 — 媒体、图片与动态取色子包（`media/` + `tokens/extract`）
> **目标**：补齐图片展示、图库预览、音乐/视频播放器与 Material You 内容动态取色能力。

- [ ] **M12.1 图像封面动态取色（`tokens/extract.go`）**
  - 纯 Go 实现轻量化 Quantizer + Score 算法（`tokens.SeedFromImage(img image.Image)`），从专辑封面/壁纸/头图提取最佳 HCT 种子色，驱动播放器与详情页的局部 `ThemeScope` 动态变色。
- [ ] **M12.2 MD3E 图像组件（根包 `image.go` + `media/netimage.go`）**
  - 根包 `md3e.Image`：支持本地位图/SVG、`ShapeScale` 或 `shape.RoundedPolygon` 异形裁切、圆角遮罩、`Fit(Cover/Contain)`、平滑淡入与加载/失败占位。
  - `media.NetImage`：封装上游 `ui/netimage`，叠加 MD3E Tonal `Skeleton` 骨架屏、失败重试占位与缓存控制。
  - `media.ImageLightbox`：点击图片平滑展开的全屏预览层（缩放/平移/左右切换/Escape 关闭）。
- [ ] **M12.3 MD3E Expressive 音视频播放器（`media/player.go`）**
  - `media.VideoPlayer`：封装上游 `ui.Video` / `ui/video`，提供 MD3E 悬浮半透明磨砂/渐变控制条（自动隐藏）、Expressive 波浪/胶囊进度拖拽条（带缓冲段与时间气泡）、异形 Play/Pause Morph 按钮、音量条、倍速菜单与全屏钩子。
  - `media.AudioPlayerCard` / `media.MiniPlayer`：紧凑底栏迷你播放器与展开式专辑封面卡片（支持封面取色 + `LinearWavyProgress` 播放波浪条）。

#### M13 — 文档、Markdown 与知识展示子包（`docview/`）
> **目标**：补齐文档站、README 预览、笔记应用、AI 流式回复渲染场景。

- [ ] **M13.1 `docview.Markdown`（封装 `ui/markdown`）**
  - 自动读取 `md3e.CurrentTheme(c)`，将标题/正文/引用/链接/分隔线/表格精准映射至 MD3E `TypographyScale` 与 `ColorScheme`。
  - 流式输出友好（适配 AI Chat 逐字/逐块更新无闪烁）。
- [ ] **M13.2 文档增强块**
  - `docview.CodeBlock`：带语言标签与一键复制按钮（带 Snackbar/勾选反馈）的 `SurfaceContainerHigh` 代码卡片。
  - `docview.Callout`：MD3E 语义化提示块（Note / Tip / Important / Warning / Caution，复用 `Primary`/`Tertiary`/`Warning`/`Error` 容器色）。
  - `docview.Outline`：基于标题结构联动滚动的右侧目录大纲（配合 `SupportingPane`）。
- [ ] **M13.3 *(可选薄桥接)* `codeview.ThemeFromMD3E`**
  - 提供极薄函数将 `*md3e.Theme` 转换为 `ui/editor` 配色与字号配置，让代码编辑器/Diff 视图开箱融入 MD3E。

#### M14 — 实时与统计图表子包（`charts/`）
> **目标**：基于上游 `ui/chart` 提供高颜值、零分配的 MD3E 数据可视化组件。

- [ ] **M14.1 仪表盘微图表**
  - `charts.AreaSparkline`：平滑曲线 + 垂直渐变面积淡出 + 环形缓冲（`RingBuffer`）复用 `ui.Path`，高频刷新零堆分配；支持无缝贴合卡片圆角底边。
  - `charts.StackedProgress`：多段水平占比条（带段间隙、圆角与自动图例，如内存/存储构成）。
  - `charts.MetricCard`：标准指标卡片（图标 + 标题 + 副指标/峰值 + 等宽大号数值 + 底部嵌入 Sparkline 或 WavyProgress）。
- [ ] **M14.2 通用统计图表（薄封装 `ui/chart`）**
  - `charts.LineChart`、`charts.BarChart`、`charts.DonutChart`：预置 HCT 谐调多序列色板（Light/Dark/HighContrast 自动可读）、悬停交互式 `RichTooltip` 数值浮标与弹簧入场过渡。

---

### P2：性能极限压榨与旗舰实战验收

#### M15 — 零分配性能治理、自动化基准与多场景旗舰 Demo
> **目标**：用真实 benchmark 数据与可运行的高颜值应用，验证“比 Flutter 更轻、更快、更美”。

- [ ] **M15.1 热路径零分配与离屏零功耗治理**
  - `shape/polygon.go` 新增 `WriteTo(dst *ui.Path, r ui.Rect)`，改造 `shape.ClipShape`、`Surface`、`Ripple`、`ShapeButton` 全部复用本地 `ui.Path`，消除每帧堆分配。
  - 全面审计 `tabs.go`、`buttongroup.go`、`ripple.go`、`list.go` 中的每帧临时切片与逃逸闭包。
  - 为 `LinearWavyProgress`、`CircularWavyProgress`、`LoadingIndicator`、`StatusPill`、`Skeleton`、`AreaSparkline` 增加 `p.Visible()` 视口相交检测，滚出可视区域后自动停止请求动画帧。
- [ ] **M15.2 性能、内存与 DCE 回归门禁（`bench_test.go`）**
  - 稳态单帧耗时与分配基准（`BenchmarkDashboardSteady`、`BenchmarkDataTable100K`、`BenchmarkSparklineTick`、`BenchmarkShapeMorphDraw`）。
  - 常驻堆内存测试（`TestRetainedHeap`），防止 Theme/Ripple/Path 缓存泄漏。
  - 自动化按需裁剪体积检查脚本/测试（确认 `Button` 最小二进制不链接 `symbols/`、`charts/`、`media/`、`docview/`）。
- [ ] **M15.3 多场景标杆参考应用（实战反哺组件库）**
  - `examples/workbench`（或拆分轻量示例）：在同一个 `AdaptiveScaffold` + `WindowTitleBar` 中内置 **3 个真实级业务页面**，覆盖 80% 场景验证：
    1. **实时监控仪表盘**（类 FlClash 风格：暖橙深色 Bento 网格、异形曲奇启停键、实时上传/下载曲线、内存分段条、万级虚拟连接/DNS 数据表、实时进程内存自显）；
    2. **媒体与图库中心**（封面动态取色 `SeedFromImage`、异形图片网格、灯箱预览、Expressive 音视频播放器 + 侧栏播放队列）；
    3. **文档与知识工作台**（左侧 `TreeView` 目录树 + 中间 `docview.Markdown` 流式文档/代码块/Callout + 右侧大纲 `Outline`）。

---

## 5. 工作量防失控机制与明确不做清单（Non-Goals）

为防止二期演进陷入“无限膨胀”的泥潭，确立以下硬边界：

### 明确不做（Non-Goals）
1. **不自研重型底层引擎**：不自写音视频解码器、不自写 Markdown/代码语法解析器、不自写图片编解码与 HTTP 缓存层、不自写复杂 3D/WebGL/地理地图图表（BI 重型图表交给业务方直接组合 `ui/chart`）。
2. **不做重型富文本所见即所得编辑器（WYSIWYG）**：只做好文档渲染（`docview`）与表单输入（`TextField`/`TextArea`），不做 Notion 级块编辑器。
3. **不强绑单一主题/产品风格**：不把“代理软件”或“暗黑暖橙”写死在组件里；所有美观度必须由 `tokens` + `Density` + `Surface` 通用能力自然派生，任何种子色在亮/暗/高对比度下都同等协调。
4. **不引入 cgo 或第三方 Go 依赖**：全线维持 `CGO_ENABLED=0` 纯 Go 与零外部第三方库（仅标准库 + `github.com/egoist/mygo`）。

### 节奏控制建议
- **第一批快速见效（M9 + M10）**：优先完成美学质感飞跃与通用微组件/图标，立即让所有现有示例脱胎换骨；
- **第二批核心场景闭环（M11 + M12 + M13 + M14）**：四个领域相互解耦，可按实际最急需的业务方向**任意排序或并行落地**；
- **收官压榨与实证（M15）**：补齐 benchmark 量化门禁与旗舰 Demo，发布 `v0.2.0` 里程碑。
