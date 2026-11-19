# plan.md — MyGo MD3E 原生组件库开发与验收总览

本文档既是全周期研发路线图，也是受 [RULE.md](RULE.md) 约束的**进度账本**。  
**强制原则**：严格按任务点编号顺序推进，**每完成 1 个点并全部通过其门禁验收，必须在同一次 `git commit` 中将对应 `- [ ]` 改为 `- [x]` 后立即提交**。

---

## 里程碑总览

| 里程碑 | 主题 | 包含任务点 | 阶段出口标准 |
|---|---|---|---|
| **M0** | 工程脚手架与验证基线 | `T0.1` | `go.mod`、`replace ./mygo`、`mygo vet` 与 `ui.NewTester` 冒烟跑通 |
| **M1** | 设计令牌与 HCT 色彩引擎 | `T1.1` – `T1.3` | HCT / DynamicScheme / Typography / Shape / Elevation 全单测全绿 |
| **M2** | 物理动效与 35 形状变形引擎 | `T2.1` – `T2.4` | 12 弹簧预设 + 35 形状 + 双射 Morph 渲染与闭合像素验收 |
| **M3** | 主题桥接、状态层与基础原语 | `T3.1` – `T3.3` | ThemeScope、Surface/Elevation、StateLayer/Ripple、内置矢量图标就绪 |
| **M4** | 行动类组件与工具栏 | `T4.1` – `T4.5` | Button/IconButton/ButtonGroup/SplitButton/FAB/FABMenu/Toolbar/AppBar |
| **M5** | 选择、滑块与输入控件 | `T5.1` – `T5.4` | Switch/Checkbox/Radio/Slider/RangeSlider/Chips/TextField |
| **M6** | 加载、波浪进度与反馈组件 | `T6.1` – `T6.3` | LoadingIndicator/WavyProgress/Badge/Tooltip/Snackbar |
| **M7** | 导航、列表、卡片与浮层 | `T7.1` – `T7.4` | Card/Divider/ExpressiveList/NavBar/NavRail/Drawer/Tabs/Carousel/Dialog/Sheet/Menu |
| **M8** | 选择器、全景画廊与总验收 | `T8.1` – `T8.3` | DatePicker/TimePicker/`examples/gallery`/全平台编译与 Golden 基线 |

---

## M0：工程脚手架与验证基线

- [x] **T0.1：初始化 Go 模块与 MyGo 本地工具链基线**
  - **交付物**：
    - `go.mod`（模块名 `mygo-md3e`，`go 1.27.3`，`tool github.com/egoist/mygo/cmd/mygo`，`require github.com/egoist/mygo v0.0.0`，`replace github.com/egoist/mygo => ./mygo`）
    - `doc.go`（根包 `md3e` 包级文档与版本信息占位）
    - `smoke_test.go`（使用 `ui.NewTester` 验证根模块构建、无头帧渲染与 `mygo vet` 生命周期检查链路）
  - **关键约束**：不修改 `mygo/` 下任何文件；必须通过 `go mod tidy` 生成干净 `go.sum`。
  - **验收标准**：
    - `CGO_ENABLED=0 go build ./...` 通过
    - `CGO_ENABLED=0 go vet ./...` 通过
    - `go tool mygo vet ./...` 通过
    - `CGO_ENABLED=0 go test ./...` 通过
    - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` 通过
  - **Commit**：`chore(repo): [T0.1] initialize go module, local mygo replace, and test harness`

---

## M1：MD3E 设计令牌与 HCT 色彩引擎（`tokens/`）

- [x] **T1.1：实现 HCT 色彩空间核心数学库**
  - **交付物**：`tokens/hct.go`、`tokens/hct_test.go`
  - **核心能力**：
    - sRGB ↔ Linear RGB ↔ CIE XYZ (D65) ↔ CIE L\*a\*b\* 精确转换；
    - CAM16 完整正算（RGB/XYZ → CAM16 `J, C, h, Q, M, s`）与 HCT 二分/解析求解反算（`HCT{Hue, Chroma, Tone}.ToColor()` 与 `HCTFromColor(c ui.Color)`）；
    - `ContrastRatio` 与 `ContrastTone`（保证合规对比度如 4.5:1 / 3:1 / 7:1 计算）。
  - **验收标准**：
    - `hct_test.go` 覆盖官方参考色值（如 `#6750A4` Baseline Purple、纯黑白灰、高饱和红绿蓝、边界 `Tone=0/100`、`Chroma=0`、`Hue` 负值/超 `360` 归一化）往返误差在可接受精度内（Tone 误差 ≤ 0.5，sRGB 通道误差 ≤ 2）；
    - `CGO_ENABLED=0 go test ./tokens/...` 与全量四重门禁通过。
  - **Commit**：`feat(tokens): [T1.1] implement HCT color space and CAM16 conversions`

- [ ] **T1.2：实现 TonalPalette、DynamicScheme 与 45+ ColorRoles**
  - **交付物**：`tokens/palette.go`、`tokens/scheme.go`、`tokens/scheme_test.go`
  - **核心能力**：
    - `TonalPalette`（支持按色阶缓存 `Tone(t float64) ui.Color`）；
    - 完整 `ColorScheme` 结构（含 [docs/md3e-spec.md](docs/md3e-spec.md) 所列全部 45+ 角色：Primary/Secondary/Tertiary 及其 Container 与 Fixed/Dim 族、Error 族、`Surface` / `SurfaceDim` / `SurfaceBright` / `SurfaceContainerLowest..Highest`、`OnSurface` / `OnSurfaceVariant`、`Outline` / `OutlineVariant`、`Inverse*`、`Scrim`、`Shadow`、`SurfaceTint`）；
    - `DynamicSchemeOptions`：支持 `Seed ui.Color`、`Dark bool`、`Variant`（`TonalSpot`, `Expressive`, `Vibrant`, `Neutral`, `Monochrome`, `Fidelity`, `Content`）、`Contrast`（`ContrastStandard`, `ContrastMedium`, `ContrastHigh`）；
    - 状态层透明度常量（Hover `0.08`, Focus `0.10`, Pressed `0.10`, Dragged `0.16`, DisabledContainer `0.12`, DisabledContent `0.38`）。
  - **验收标准**：
    - 验证 Light / Dark 下 `SurfaceContainerLowest..Highest` 亮度严格单调（Light 从 100 递减至 90；Dark 从 4 递增至 22）；
    - 验证 `PrimaryFixed` / `OnPrimaryFixed` 在同种子 Light 与 Dark 模式下数值一致；
    - 验证 `ContrastHigh` 比 `ContrastStandard` 下 `OnPrimary` 与 `Primary`、`OnSurface` 与 `Surface` 对比度严格更高且满足 AA/AAA 阈值。
  - **Commit**：`feat(tokens): [T1.2] implement TonalPalette, DynamicScheme variants, and MD3 color roles`

- [ ] **T1.3：实现 Shape、SizeTier、Typography（含 Emphasized）与 Elevation Tokens**
  - **交付物**：`tokens/shape.go`、`tokens/size.go`、`tokens/type.go`、`tokens/elevation.go`、`tokens/tokens_test.go`
  - **核心能力**：
    - `ShapeScale` 10 档圆角（`None`, `ExtraSmall`, `Small`, `Medium`, `Large`, `LargeIncreased`, `ExtraLarge`, `ExtraLargeIncreased`, `ExtraExtraLarge`, `Full`）；
    - `SizeTier` 5 档尺寸（`SizeXS`, `SizeS`, `SizeM`, `SizeL`, `SizeXL`）精确参数表；
    - `TypographyScale`（Standard 与 Emphasized 共 30 个角色样式）与 `TypeRole` 枚举；
    - `ElevationLevel`（`Level0..Level5`：dp 高度、阴影参数、Tonal Tint 混合比例）。
  - **验收标准**：
    - 单测覆盖全部 5 档 SizeTier、10 档圆角、30 组排版令牌非零有效性及 Level 0–5 阴影参数单调性；
    - 四重门禁全绿。
  - **Commit**：`feat(tokens): [T1.3] add shape scale, XS-XL size tiers, emphasized typography, and elevation tokens`

---

## M2：MD3E 物理动效与 35 种几何变形引擎（`motion/` & `shape/`）

- [ ] **T2.1：实现 MD3E Spring Physics 动效系统（`motion/`）**
  - **交付物**：`motion/motion.go`、`motion/motion_test.go`
  - **核心能力**：
    - `Scheme`（`Expressive` / `Standard`）、`Token`（`FastSpatial`, `DefaultSpatial`, `SlowSpatial`, `FastEffects`, `DefaultEffects`, `SlowEffects`）；
    - `SpringSpec` 构造函数：正确实现 $c = 2 \cdot \zeta \cdot \sqrt{k}$ ($m=1$) 转换到 `ui.SpringOptions{Mass: 1, Stiffness: k, Damping: c}`；
    - Spatial 允许按规范过冲，Effects 强制临界阻尼（$\zeta=1.0$）、`Oklab: true`、`NonMotion: true`；
    - 提供 `Resolve(scheme Scheme, token Token, reduceMotion bool) ui.Animation` 与便捷动画辅助方法。
  - **验收标准**：
    - 数学单测检验 12 组预设 `SpringOptions.Progress(t)` 行为：`Expressive FastSpatial` 存在明显过冲（max > 1.04）；`Effects` 任意时间点无过冲（0 ≤ Progress ≤ 1.001）；
    - 在 `ui.NewTester` 中驱动实际动画元素：推进 `SettleTime()` + 裕量后 `tt.FramePending() == false`，且 `ReduceMotion: true` 时 0 延迟收敛且不产生额外动画帧。
  - **Commit**：`feat(motion): [T2.1] implement MD3E spring physics schemes and damping conversion`

- [ ] **T2.2：实现 RoundedPolygon 几何基元与 Cubic Bézier 归一化（`shape/`）**
  - **交付物**：`shape/polygon.go`、`shape/polygon_test.go`
  - **核心能力**：
    - `CornerRounding{Radius, Smoothing}`（含平滑圆角曲率过渡，支持 `Unrounded`）；
    - 从正多边形（`RegularPolygon`）、星形（`Star`）、矩形（`Rectangle`）或任意带独立圆角顶点列表（`NewPolygon(vertices, perVertexRounding)`）生成闭合 `Cubic` 曲线序列；
    - 几何变换（`Transformed` / 平移 / 缩放 / 旋转）、归一化至单位框（`[0,1]×[0,1]` 或 `[-1,1]`）、包围盒计算及导出为 `*ui.Path`（`ToPath(r ui.Rect)`）。
  - **验收标准**：
    - 单测验证正多边形、星形、圆角矩形导出 `*ui.Path` 的闭合性、边界不越界与 `Path.Contains(center)` 命中，结合 `ui.Render` 验证四象限像素填充覆盖正确；
    - 四重门禁全绿。
  - **Commit**：`feat(shape): [T2.2] implement RoundedPolygon geometry, corner smoothing, and ui.Path builder`

- [ ] **T2.3：实现 MD3E 35 种内置形状目录（`shape/catalog.go`）**
  - **交付物**：`shape/catalog.go`、`shape/catalog_test.go`
  - **核心能力**：
    - 完整实现 [docs/md3e-spec.md](docs/md3e-spec.md) 列出的 35 种内置形状（`Circle` 到 `Heart`），并提供统一枚举/名称目录 `All()` 便于遍历与画廊展示；
    - 每种形状具备合理的特征标记（用于后续 Morph 对齐）且几何边界归一化在单位正方形 `[0, 1]×[0, 1]` 内。
  - **验收标准**：
    - 单测遍历全部 35 种形状（核验数量 = 35、名称唯一、非空 `Cubic` 曲线、全部控制点与锚点在合理边界内）；
    - 使用 `ui.Render` 逐一渲染 35 种形状到 `64×64` 画布，断言每种形状均有有效前景像素且零像素溢出到裁切框外。
  - **Commit**：`feat(shape): [T2.3] implement all 35 Material 3 Expressive shapes catalog`

- [ ] **T2.4：实现任意双形状双射变形器（`shape/morph.go`）与绘制辅助**
  - **交付物**：`shape/morph.go`、`shape/draw.go`、`shape/morph_test.go`
  - **核心能力**：
    - `NewMorph(from, to *RoundedPolygon) *Morph`：按极角进度映射（Angle/Progress DoubleMapper）对两条闭合 Cubic 链做等细分切割，保证段数一致且角点特征尽量对齐；
    - `morph.AsPath(progress float32, dst ui.Rect) *ui.Path` 与 `DrawMorph` / `ClipShape`；
    - 支持静态复用零多余分配优化。
  - **验收标准**：
    - 对全部 35 种形状两两组合抽样（含凸形↔凹形如 `Circle↔SoftBurst`、`Square↔Heart`、`Pentagon↔Sunny`、`Pill↔Cookie9Sided`）：
      - `progress = 0` 与 `from` 绘制图像一致、`progress = 1` 与 `to` 绘制图像一致；
      - `progress = 0.25, 0.5, 0.75` 渲染非空、包围盒合理、曲线连续无 NaN/Inf、无边界外溢出；
    - 四重门禁全绿。
  - **Commit**：`feat(shape): [T2.4] implement bidirectional cubic morphing between arbitrary MD3E shapes`

---

## M3：主题桥接、状态层、涟漪与基础原语（根包 `md3e`）

- [ ] **T3.1：实现 `md3e.Theme`、上下文作用域与 `ui.Theme` 桥接**
  - **交付物**：`theme.go`、`theme_test.go`
  - **核心能力**：
    - `Theme` 结构（聚合 `tokens.ColorScheme`、`tokens.TypographyScale`、`tokens.ShapeScale`、`motion.Scheme`、`IsDark bool`、`Contrast`、以及动态派生配置）；
    - `ThemeFromSeed(seed ui.Color, opts ...ThemeOptions) *Theme`、`DefaultTheme(c *ui.Context) *Theme`（根据 `c.Theme().Dark` 与 `c.Preferences()` 自动缓存派生）；
    - `ThemeScope(c *ui.Context, t *Theme, fn func()) ui.Element`（同步设置 `ui.Themed(c, t.ToUITheme())` 并将 `*md3e.Theme` 挂载在作用域中，兼容 `ui.Overlay` 继承）；
    - `CurrentTheme(c *ui.Context) *Theme`。
  - **验收标准**：
    - 测试验证嵌套 `ThemeScope`（外层 Light 紫、内层 Dark 绿）下，内外层及内部弹出的 `ui.Overlay` 各自读取到正确的 `CurrentTheme(c)`；
    - 测试验证 `tt.SetDark(true)`、`tt.SetPreferences`（`HighContrast`, `Accent`）变更后默认主题自动响应切换。
  - **Commit**：`feat(theme): [T3.1] implement md3e.Theme, ThemeScope, and ui.Theme/Preferences bridge`

- [ ] **T3.2：实现 Surface、Elevation 渲染、Typography 文本与 StateLayer/Ripple**
  - **交付物**：`surface.go`、`ripple.go`、`typography.go`、`surface_test.go`
  - **核心能力**：
    - `Surface` 原语：支持背景色、Tonal Elevation 色调叠加、MD3E Elevation `Level0..5` 多层阴影（Key + Ambient shadow）、边框、独立四角半径；
    - `Text(c *ui.Context, s string, role tokens.TypeRole, emphasized ...bool) ui.Element`；
    - `StateLayer` 与 `ExpressiveRipple`：使用 `ui.Local` 跟踪按压起点（中心或指针坐标）、扩散与淡出动画（仅在 `DrawOver` 中读取 `Hovered()` / `Pressed()`，避免整树重建）；遵循 `ReduceMotion`（禁用波纹扩散改用即时静态态层）；支持独立四圆角裁切。
  - **验收标准**：
    - `surface_test.go` 使用 `ui.NewTester` 验证：Hover 时产生 8% 态层变化、Press 时产生指针起点波纹像素扩散、松手后推进时钟至稳定状态 `tt.FramePending() == false`、开启 `ReduceMotion` 无空转帧；
    - 四重门禁全绿。
  - **Commit**：`feat(core): [T3.2] implement Surface, Elevation, Typography helper, StateLayer, and Ripple`

- [ ] **T3.3：构建内置基础矢量图标集（`internal/mdicon`）**
  - **交付物**：`internal/mdicon/icons.go`、`internal/mdicon/icons_test.go`
  - **核心能力**：
    - 提供组件库自身所需的标准 Material Symbols 24×24 viewBox 矢量路径/SVG（如 `Check`, `Minus`, `Close`, `ChevronDown`, `ChevronRight`, `ArrowBack`, `ArrowForward`, `Search`, `MoreVert`, `MoreHoriz`, `Add`, `Edit`, `Calendar`, `Clock`, `Visibility`, `VisibilityOff`, `ErrorOutline`, `Star`）；
    - 轻量按需解析（独立变量，不打包巨大全局 slice 导致未用图标被强链接）。
  - **验收标准**：
    - 所有内置图标可无 panic 解析、包围盒规范在 `[0, 24]` 附近，且 16×16/24×24 渲染像素非空；
    - 四重门禁全绿。
  - **Commit**：`feat(icons): [T3.3] add internal built-in Material vector icons for component chrome`

---

## M4：行动类核心组件与工具栏

- [ ] **T4.1：实现 Expressive Button（5 变体 × 5 尺寸 × 形态形变）**
  - **交付物**：`button.go`、`button_test.go`
  - **核心能力**：
    - 基于 `ui.ButtonBase` 构建，支持 `ButtonFilled`, `ButtonTonal`, `ButtonElevated`, `ButtonOutlined`, `ButtonText`；
    - 支持 `SizeXS..SizeXL` 与 `ButtonShapeRound` / `ButtonShapeSquare`；
    - 按压时圆角使用 `motion.FastSpatial` 向按压态圆角形变（松手回弹）；支持可选前置/后置图标槽位、禁用态精准色值（`DimDisabled(false)`）、StateLayer 与 Ripple、完整无障碍角色与标签。
  - **验收标准**：
    - 测试覆盖 5 种变体 × 5 种尺寸渲染高度误差 0、点击/Enter/Space 触发、按压圆角收缩动画及释放收敛、禁用态不响应点击与按键、`AccessByRole(ui.RoleButton, ...)` 检查；
    - 四重门禁全绿。
  - **Commit**：`feat(button): [T4.1] implement Expressive Button with 5 variants, XS-XL sizes, and corner morph`

- [ ] **T4.2：实现 Expressive IconButton & ToggleIconButton**
  - **交付物**：`iconbutton.go`、`iconbutton_test.go`
  - **核心能力**：
    - 基于 `ui.ButtonBase` / `ui.ToggleBase`，支持 `IconButtonStandard`, `IconButtonFilled`, `IconButtonTonal`, `IconButtonOutlined`；
    - 支持 `SizeXS..SizeXL` 与 Width 规格（Narrow / Default / Wide）；
    - `ToggleIconButton` 在未选中（圆形）与选中（方形圆角）之间用 `FastSpatial` 弹簧平滑切换形态，且暴露 `Checked` 无障碍状态。
  - **验收标准**：
    - 测试验证普通点击与 Toggle 选中切换时边界圆角变化、键盘 Space 切换、`Label` 无障碍命名强制可用、Narrow/Default/Wide 宽度差异准确。
  - **Commit**：`feat(iconbutton): [T4.2] implement Expressive IconButton and morphing ToggleIconButton`

- [ ] **T4.3：实现 ButtonGroup（Connected / Standard）与 SplitButton**
  - **交付物**：`buttongroup.go`、`splitbutton.go`、`buttongroup_test.go`、`splitbutton_test.go`
  - **核心能力**：
    - **ButtonGroup**：支持 `ButtonGroupStandard`（按压/选中按钮沿主轴弹性扩展，轻微挤压邻近按钮）与 `ButtonGroupConnected`（首项外侧圆角大、内部接缝小圆角、选中项切换全圆角）；支持单选/多选/动作组模式，键盘方向键焦点导航；
    - **SplitButton**：基于 `ui.Segmented`/焦点组模型（对齐 `ui.SplitButtonParts`：`Group`, `Button`, `Arrow`），支持 `Filled`/`Tonal`/`Elevated`/`Outlined` 与 `XS..XL`；展开下拉菜单时 Arrow 段旋转 180° 并将非对称内接圆角形变为对称全圆角，支持 `Alt+Down` / `Space` / `Enter` 无障碍交互与 RTL 镜像。
  - **验收标准**：
    - `buttongroup_test.go` 验证 Standard 挤压时按压按钮宽度增大而邻项收敛、`Connected` 模式选中项圆角变化及方向键导航；
    - `splitbutton_test.go` 验证主动作点击与 Arrow 展开/收起状态、RTL 下的左右圆角镜像及无障碍角色树。
  - **Commit**：`feat(actions): [T4.3] implement Expressive ButtonGroup and morphing SplitButton`

- [ ] **T4.4：实现 FAB、ExtendedFAB 与 FABMenu**
  - **交付物**：`fab.go`、`fabmenu.go`、`fab_test.go`
  - **核心能力**：
    - `FAB`（`SizeSmall` 40 / `SizeMedium` 56 / `SizeLarge` 96）与 `ExtendedFAB`（支持可折叠展开标签）；支持 `PrimaryContainer`/`SecondaryContainer`/`TertiaryContainer`/`Surface` 配色方案；
    - `FABMenu`（符合 MD3E 规范取代 Speed Dial）：仅从标准 FAB 展开 **2–6 个**高对比大圆角菜单项（`FABMenuItem`），带 `Overlay` + `Scrim` 遮罩，原 FAB 变形切换为圆形关闭按钮（`×` 旋转动画），Escape / 点击遮罩 / 选中项自动关闭并归还焦点。
  - **验收标准**：
    - 测试验证 FAB 三种尺寸精确像素框、ExtendedFAB 折叠/展开宽度变化、FABMenu 项数校验（<2 或 >6 明确约束/测试边界）、展开后菜单项点击回调、Escape 关闭与焦点归还。
  - **Commit**：`feat(fab): [T4.4] implement FAB, ExtendedFAB, and Expressive FABMenu`

- [ ] **T4.5：实现 FloatingToolbar 与 AppBar（TopAppBar / SearchAppBar）**
  - **交付物**：`toolbar.go`、`appbar.go`、`toolbar_test.go`
  - **核心能力**：
    - `FloatingToolbar`：支持 `Horizontal` 与 `Vertical` 方向、`Standard` 与 `Vibrant` 双色调主题、紧邻耦合标准 `FAB` 布局及显隐/折叠弹簧动画；
    - `TopAppBar`（`Small` / `Medium` / `Large` / `CenterAligned`，支持副标题与滚动收起进度 `scrollProgress` 高度及字号插值）与 `SearchAppBar`。
  - **验收标准**：
    - 测试验证 Horizontal/Vertical 工具栏布局尺寸、耦合 FAB 点击互不遮挡、TopAppBar 四种变体与滚动收起高度平滑过渡、SearchAppBar 输入与提交回调。
  - **Commit**：`feat(bars): [T4.5] implement FloatingToolbar and Expressive TopAppBar/SearchAppBar`

---

## M5：选择、滑块与输入控件

- [ ] **T5.1：实现 Expressive Switch、Checkbox 与 Radio**
  - **交付物**：`switch.go`、`checkbox.go`、`radio.go`、`selection_controls_test.go`
  - **核心能力**：
    - `Switch`（基于 `ui.SwitchBase`）：`52×32` 轨道，拇指在 Off(`16`/`24`)、On(`24` + `✓`)、Pressed(`28`) 间通过 `FastSpatial` 弹簧缩放平移，支持 RTL；
    - `Checkbox` & `CheckboxState`（基于 `ui.CheckboxBase` / `ui.CheckboxStateBase`）：支持 Off / On / Mixed 三态，勾选与连字符由 `ui.Path.Trim` 弹性绘制，支持错误态配色；
    - `Radio` & `RadioGroup`（基于 `ui.RadioBase`）：外环与内圆缩放动画及完整方向键组导航。
  - **验收标准**：
    - 覆盖点击/Space 切换、Switch 拇指尺寸在选中前后对比、Checkbox 三态切换、RadioGroup 方向键单选、Disabled 与 `ReduceMotion`、`AccessByRole` 检查。
  - **Commit**：`feat(selection): [T5.1] implement Expressive Switch, tri-state Checkbox, and RadioGroup`

- [ ] **T5.2：实现 Expressive Slider & RangeSlider**
  - **交付物**：`slider.go`、`slider_test.go`
  - **核心能力**：
    - `Slider`（基于 `ui.SliderBase`）与 `RangeSlider`（双拇指焦点/拖拽）：
      - 竖直胶囊拇指（`4×44`），与轨道之间保留 `6` DIP 间隙（Track Gap）；
      - 离散 `Step` 显示轨道停靠圆点（Stop Indicators）；
      - 拖拽或键盘聚焦激活时在拇指上方弹出水滴形（Teardrop）值标签气泡；
      - 支持 `Centered` 模式（以中间基准点向左右/正负双向填充）及 RTL 镜像。
  - **验收标准**：
    - 测试覆盖拖动滑块、左右/Home/End 键盘步进、RangeSlider 两端拇指不越界交叉、停靠点与气泡显示隐藏动画收敛、RTL 方向值增减反转、无障碍 `Range` 值断言。
  - **Commit**：`feat(slider): [T5.2] implement Expressive Slider and RangeSlider with gap, stops, and value indicator`

- [ ] **T5.3：实现 Assist / Filter / Input / Suggestion Chips**
  - **交付物**：`chip.go`、`chip_test.go`
  - **核心能力**：
    - 四种 MD3E Chip：`AssistChip`、`FilterChip`（选中态勾选图标入场、容器背景色与圆角切换）、`InputChip`（可附带头像与可点击关闭按钮 `×`）、`SuggestionChip`；
    - 支持 `Elevated` 与默认描边两种外观，提供 `ChipGroup`（自动换行 `Wrap()` 容器）。
  - **验收标准**：
    - 测试覆盖 FilterChip 选中态切换与勾选绘制、InputChip 点击主体与点击删除按钮的独立命中区分、Disabled 拦截与无障碍名称/选中态验证。
  - **Commit**：`feat(chip): [T5.3] implement Assist, Filter, Input, and Suggestion Chips`

- [ ] **T5.4：实现 Expressive TextField & TextArea（Filled / Outlined）**
  - **交付物**：`textfield.go`、`textfield_test.go`
  - **核心能力**：
    - 基于 `ui.TextInputBase` / `ui.TextAreaBase` 构建；
    - 支持 `TextFieldFilled`（顶部圆角 + 底部指示线 + 填充底色）与 `TextFieldOutlined`（完整外框 + 标签上浮时上边框自动断开缺口）；
    - 浮动标签（空且失焦居于内容区，有字或聚焦时弹簧缩放到顶部）；
    - 支持 Prefix/Suffix、Leading/Trailing 图标插槽、SupportingText、CharacterCounter、Error 错误态（配色与 `Invalid`/`ErrorMessage` 无障碍关联）、ReadOnly、Password。
  - **验收标准**：
    - 测试覆盖聚焦→标签上浮→输入文本→提交→失焦、Outlined 缺口随文本非空保持、Error 状态无障碍 `Invalid` 与 `ErrorMessage` 检出、IME 组合态兼容、多行 TextArea 换行与滚动。
  - **Commit**：`feat(textfield): [T5.4] implement Filled and Outlined TextField with floating label and error states`

---

## M6：进度、加载与反馈组件

- [ ] **T6.1：实现 Expressive Shape-Morphing `LoadingIndicator`**
  - **交付物**：`loading.go`、`loading_test.go`
  - **核心能力**：
    - 基于 `shape.Morph` 与 `motion` 弹簧驱动，在可配置多边形序列（默认 7 种 MD3E 经典形状循环）之间连续变形并伴随角速度旋转；
    - 支持 `Standard`（透明底 + `Primary` 图形）与 `Contained`（圆形 `PrimaryContainer` 底 + `OnPrimaryContainer` 图形），可配置尺寸、自定义形状序列、`Paused`；
    - 开启 `ReduceMotion` 时保持静止且 `tt.FramePending() == false`。
  - **验收标准**：
    - 测试验证不同时间点（如 `0ms`, `300ms`, `650ms`）渲染帧像素明显变化；
    - 测试验证所有阶段绘制像素严格位于组件边界内；
    - 测试验证 `ReduceMotion` 与 `Paused` 下零空转帧且前后帧像素完全一致；
    - 无障碍角色 `RoleProgress` 检出。
  - **Commit**：`feat(loading): [T6.1] implement shape-morphing MD3E LoadingIndicator`

- [ ] **T6.2：实现 Wavy 与标准 Linear / Circular 进度指示器**
  - **交付物**：`progress.go`、`progress_test.go`
  - **核心能力**：
    - `LinearProgress` & `LinearWavyProgress`（确定态 `0..1` 与不确定态 `< 0`）：正弦波浪活跃段、端点平滑振幅收敛（接近 `0` 或 `1` 时渐平）、Track Gap 间隙与末端停靠圆点、支持 RTL；
    - `CircularProgress` & `CircularWavyProgress`：极坐标波动弧（Determinate 弧长动画 / Indeterminate 弧长伸缩+连续旋转）；
    - `ReduceMotion` 时不定态展示合规静止样式并停止空转帧。
  - **验收标准**：
    - 确定态 `0.0`, `0.5`, `1.0` 与不确定态多时间点像素采样验证活跃比例与波浪推进；
    - RTL 下 `LinearWavyProgress` 从右侧开始填充；
    - 无障碍 `RoleProgress` 范围 `[0, 1, value]` 校验。
  - **Commit**：`feat(progress): [T6.2] implement Linear/Circular and Wavy ProgressIndicators`

- [ ] **T6.3：实现 Badge、Tooltip（Plain & Rich）与 Snackbar**
  - **交付物**：`badge.go`、`tooltip.go`、`snackbar.go`、`feedback_test.go`
  - **核心能力**：
    - `Badge`（Small `6×6` 红点与 Large 带数字/上限截断如 `999+` 徽章）与 `BadgedBox` 锚定辅助；
    - `PlainTooltip` & `RichTooltip`（基于 `ui.TooltipBase` / `ui.PopoverBase`：支持标题、正文与操作按钮，Hover/Focus 延迟触发与 Escape 隐藏）；
    - `Snackbar` & `SnackbarHost`（基于 `ui.Overlay` 与单队列状态 `SnackbarState`：支持单行/双行、可选 Action 按钮、Close 按钮、超时自动消失及手势滑走，悬停时暂停倒计时）。
  - **验收标准**：
    - 测试验证 `Badge` 数字格式化与空值/红点尺寸；
    - 测试验证 `Tooltip` 在推进延迟后显现、Escape 关闭；
    - 测试验证 `Snackbar` 显示、点击 Action 回调、悬停暂停倒计时、超时自动退出队列且动画结束后 `tt.FramePending() == false`。
  - **Commit**：`feat(feedback): [T6.3] implement Badge, Plain/Rich Tooltip, and Snackbar queue`

---

## M7：导航、列表、卡片与浮层组件

- [ ] **T7.1：实现 Cards、Divider 与 Expressive Segmented List**
  - **交付物**：`card.go`、`divider.go`、`list.go`、`container_test.go`
  - **核心能力**：
    - `ElevatedCard`、`FilledCard`、`OutlinedCard`（含可点击态涟漪/海拔浮起与不可点击容器态）；
    - `Divider`（Horizontal / Vertical，支持 `Inset` 左右缩进）；
    - `List` & `ListItem`（支持 Expressive **分段圆角模式**：首项顶部大圆角、中间项小圆角、末项底部大圆角，选中或按压时独立向大圆角平滑动画；支持 Leading Avatar/Icon/Checkbox、Headline、SupportingText、Trailing Meta/Control 槽位）。
  - **验收标准**：
    - 测试验证三种 Card 外观与点击态、Divider 水平/垂直与缩进、Expressive List 多项分组圆角配置及选中态变化、无障碍列表项角色与选中状态。
  - **Commit**：`feat(containers): [T7.1] implement MD3E Cards, Divider, and Expressive Segmented List`

- [ ] **T7.2：实现 NavigationBar、NavigationRail 与 NavigationDrawer**
  - **交付物**：`navbar.go`、`navrail.go`、`navdrawer.go`、`navigation_test.go`
  - **核心能力**：
    - `NavigationBar`：底部目的地导航，选中项弹性胶囊指示器（宽度与透明度弹簧过渡）、图标+标签（Always / SelectedOnly / Never）及内置 `Badge`；
    - `NavigationRail`：支持 `Collapsed`（紧凑图标+短标签）与 `Expanded`（MD3E 展开为带左侧圆角药丸高亮的扩展侧栏，平滑过渡宽度），支持头部 FAB 槽位；
    - `NavigationDrawer`（Modal 抽屉遮罩+拖拽关闭 & Standard/Dismissible 常驻抽屉）：支持分段标题、Divider、活跃药丸胶囊及 Badge 文字。
  - **验收标准**：
    - 测试覆盖点击切换 `*selected` 更新与回调触发、NavigationRail 折叠↔展开宽度动画与选项选择、Modal Drawer 遮罩点击/Escape/拖拽关闭与焦点管理。
  - **Commit**：`feat(navigation): [T7.2] implement NavigationBar, expandable NavigationRail, and NavigationDrawer`

- [ ] **T7.3：实现 Primary/Secondary Tabs 与 Expressive Carousel**
  - **交付物**：`tabs.go`、`carousel.go`、`tabs_carousel_test.go`
  - **核心能力**：
    - `PrimaryTabs` & `SecondaryTabs`（基于 `ui.TabsBase`）：Primary 带底部圆角指示条且支持“指示条跟随位置平滑滑动”（通过保存前一选中几何位置做弹簧过渡）与图标+标签；Secondary 贴底全宽线；支持固定等分与横向可滚动溢出标签；
    - `Carousel`：支持 `Hero`（主焦点大图 + 邻近小图裁切）、`MultiBrowse`（大-中-小多档渐变宽度）与 `Uncontained` 三种策略，支持横向手势/滚轮滑动及吸附、项内 `ClipInset` 圆角动态宽度变化。
  - **验收标准**：
    - `Tabs` 测试覆盖点击与左右方向键切换、滑动指示条收敛、滚动 Tabs；
    - `Carousel` 测试覆盖拖拽切换主焦点项后各槽位尺寸动态重分配并最终停止动画。
  - **Commit**：`feat(nav-media): [T7.3] implement sliding-indicator Tabs and Expressive Carousel`

- [ ] **T7.4：实现 Dialog、BottomSheet 与 Expressive Menu**
  - **交付物**：`dialog.go`、`sheet.go`、`menu.go`、`overlay_test.go`
  - **核心能力**：
    - `AlertDialog`（基于 `ui.DialogBase`：图标槽、大标题、正文、可滚动内容、Confirm/Dismiss 按钮区、弹簧缩放淡入淡出）与 `FullScreenDialog`（顶栏关闭与保存动作）；
    - `BottomSheet`（Modal & Standard，支持 `Detents` 停靠高度、顶部拖拽把手、下拉关闭手势、`Scrim` 渐隐及内嵌可滚动内容不冲突）；
    - `DropdownMenu` / `MenuButton` / `ContextMenu`（基于 `ui.MenuBase` / `ui.ContextMenuBase`：分层圆角菜单面板、`PrimaryContainer`/`SecondaryContainer` 分组高亮、图标、快捷键文本、勾选/单选/子菜单项）。
  - **验收标准**：
    - 测试验证 Dialog 焦点锁定与 Escape/遮罩关闭、BottomSheet 手势拖至关闭档位及停靠回弹、Menu 键盘上下移动/Enter 选中/Submenu 展开。
  - **Commit**：`feat(overlay): [T7.4] implement Alert/FullScreen Dialog, BottomSheet, and Expressive Menu`

---

## M8：高级选择器、全量演示画廊与质量总验收

- [ ] **T8.1：实现 DatePicker 与 TimePicker**
  - **交付物**：`datepicker.go`、`timepicker.go`、`picker_test.go`
  - **核心能力**：
    - `DatePicker`（Modal 弹窗与 Docked 内联两种形态）：月历网格、今天圆圈标识、选中日期胶囊高亮、支持单选与日期范围选（RangeHighlight）、切换月份/年份视图及切换文本输入模式；
    - `TimePicker`（钟盘 Dial 与数字 Input 双模式）：12/24 小时制切换、AM/PM 分段开关、表盘指针与角度拖拽选时分（选择小时后自动平滑过渡到分钟盘）。
  - **验收标准**：
    - `picker_test.go` 覆盖点击选日、跨月翻页、日期范围跨天高亮、表盘选时→自动切分及键盘/输入框模式同步、完整无障碍网格/按钮角色检验。
  - **Commit**：`feat(pickers): [T8.1] implement DatePicker and TimePicker`

- [ ] **T8.2：构建交互式 MD3E 全量演示画廊（`examples/gallery`）**
  - **交付物**：`examples/gallery/`（多页面 `ui.Router` 应用，按分类展示所有 M1–M8 组件）
  - **核心能力**：
    - 顶部/侧边切换实时 Seed Color、DynamicScheme Variant（TonalSpot/Expressive/Vibrant...）、Light/Dark、Contrast（Standard/Medium/High）、MotionScheme（Expressive/Standard）、RTL 切换；
    - 专门页面交互演示：35 形状实时两两 Morph 游乐场、LoadingIndicator & WavyProgress 控制台、ButtonGroup/SplitButton/FABMenu/Toolbar 交互区、Slider/Inputs/Pickers/Navigation/Overlays 实验室；
    - 含 `examples/gallery/gallery_test.go`（使用 `ui.NewTester` 遍历各画廊路由页面点击交互，确保零 panic、零 stale element、零未命名可点击按钮）。
  - **验收标准**：
    - `CGO_ENABLED=0 go test ./examples/gallery/...` 通过；
    - `CGO_ENABLED=0 go build ./examples/gallery` 生成无报错（验证可执行构建，切记不提交 `.exe`！）；
    - `go tool mygo vet ./...` 全部通过。
  - **Commit**：`feat(gallery): [T8.2] add interactive MD3E showcase gallery application and smoke tests`

- [ ] **T8.3：全仓最终质量验收、Golden 基线与使用文档**
  - **交付物**：`README.md`、核心组件代表性 `ui/golden` 测试（或跨平台校验脚本/测试补全）
  - **核心能力**：
    - 完善根目录 `README.md`：包含安装/本地 `replace` 用法、快速上手代码示例、组件总表、主题与动效配置说明、测试指引；
    - 补齐跨平台、RTL、`ReduceMotion` 与边界回归检查。
  - **验收标准**（全仓最高标准终验）：
    - `CGO_ENABLED=0 go build ./...` 通过
    - `CGO_ENABLED=0 go vet ./...` 通过
    - `go tool mygo vet ./...` 通过
    - `CGO_ENABLED=0 go test ./...` 全绿（无跳过/无放宽）
    - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` 通过
    - `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./...` 通过
    - `GOOS=js GOARCH=wasm go build ./...` 通过
    - `git status --porcelain` 干净，无遗留二进制/临时截图产物。
  - **Commit**：`docs(repo): [T8.3] add README, final cross-platform verification, and release baseline`
