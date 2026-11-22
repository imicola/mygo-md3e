# Material Design 3 Expressive (MD3E) 技术规格基线

本规格文档为 `mygo-md3e` 提供精确的数值基准与交互行为契约，确保实现忠于 **Google Material Design 3 Expressive** 规范，避免与旧版 M2 或早期 M3 规格混淆。

---

## 1. 色彩系统（HCT & Color Roles）

### 1.1 HCT 色彩空间与 Tonal Palette
- 色彩空间：**HCT**（Hue `0..360°`，Chroma `>= 0`，Tone `0..100`），基于 **CAM16** 色貌模型（Hue & Chroma）+ **CIE L\*a\*b\*** 感知亮度（Tone = L\*）。
- **Tonal Palette** 核心色阶：`0, 4, 6, 10, 12, 17, 20, 22, 24, 25, 30, 35, 40, 50, 60, 70, 80, 87, 90, 92, 94, 95, 96, 98, 99, 100`（支持任意 Tone `0..100` 动态求值）。
- 支持动态派生方案（`Variant`）：`TonalSpot`（默认）、`Expressive`（强调色偏转更具表现力）、`Vibrant`、`Neutral`、`Monochrome`、`Fidelity`、`Content`；支持三档对比度：`Standard (0.0)`、`Medium (0.5)`、`High (1.0)`。

### 1.2 45+ 语义色彩角色（`ColorScheme`）
必须完整定义以下角色（不可缺省 Surface Container 五层级与 Fixed 角色）：
- **Primary 族**：`Primary`, `OnPrimary`, `PrimaryContainer`, `OnPrimaryContainer`, `InversePrimary`, `PrimaryFixed`, `PrimaryFixedDim`, `OnPrimaryFixed`, `OnPrimaryFixedVariant`
- **Secondary 族**：`Secondary`, `OnSecondary`, `SecondaryContainer`, `OnSecondaryContainer`, `SecondaryFixed`, `SecondaryFixedDim`, `OnSecondaryFixed`, `OnSecondaryFixedVariant`
- **Tertiary 族**：`Tertiary`, `OnTertiary`, `TertiaryContainer`, `OnTertiaryContainer`, `TertiaryFixed`, `TertiaryFixedDim`, `OnTertiaryFixed`, `OnTertiaryFixedVariant`
- **Error 族**：`Error`, `OnError`, `ErrorContainer`, `OnErrorContainer`
- **Surface 与 Neutral 族**：
  - `Background`, `OnBackground`
  - `Surface`, `SurfaceDim`, `SurfaceBright`, `OnSurface`, `OnSurfaceVariant`
  - `SurfaceContainerLowest`（Light: Tone 100 / Dark: Tone 4）
  - `SurfaceContainerLow`（Light: Tone 96 / Dark: Tone 10）
  - `SurfaceContainer`（Light: Tone 94 / Dark: Tone 12）
  - `SurfaceContainerHigh`（Light: Tone 92 / Dark: Tone 17）
  - `SurfaceContainerHighest`（Light: Tone 90 / Dark: Tone 22）
  - `SurfaceTint`（通常同 `Primary`）
- **Outline & Utility**：`Outline`, `OutlineVariant`, `InverseSurface`, `InverseOnSurface`, `Scrim`, `Shadow`

### 1.3 状态层（State Layers）与禁用透明度
- **State Layer 透明度**（以内容前景色或指定态层色叠加在容器之上）：
  - Hover: `0.08`
  - Focus: `0.10`
  - Pressed: `0.10`（配合 Expressive Ripple 波纹扩散）
  - Dragged: `0.16`
- **Disabled 状态**：
  - 禁用容器底色透明度：`0.12`（叠加 `OnSurface`）
  - 禁用文本/图标前景色透明度：`0.38`（使用 `OnSurface`）
  - 注：使用 `ui.Element.DimDisabled(false)` 自行精准呈现 MD3E 禁用色值，避免 `ui` 默认半透明整体变淡失真。

---

## 2. 物理动效系统（MD3E Motion Physics）

### 2.1 12 组精确弹簧令牌
MD3E 将动效分为 **Spatial（空间：位移、缩放、尺寸、圆角、形状 Morph，允许弹性过冲）** 与 **Effects（视觉效果：颜色、透明度、状态层，临界阻尼无过冲）**，提供 **Expressive**（默认表现力方案）与 **Standard**（克制工具方案）：

| Scheme | 类别 | 速度 | 阻尼比 ($\zeta$) | 刚度 ($k$) | 换算公式 ($m=1$ 时 $c = 2\zeta\sqrt{k}$) | 约略稳定时间 |
|---|---|---|---|---|---|---|
| **Expressive** | Spatial | Fast | `0.6` | `800` | `Damping ≈ 33.9411` | ~420 ms |
| **Expressive** | Spatial | Default | `0.8` | `380` | `Damping ≈ 31.1897` | ~475 ms |
| **Expressive** | Spatial | Slow | `0.8` | `200` | `Damping ≈ 22.6274` | ~655 ms |
| **Standard** | Spatial | Fast | `0.9` | `1400` | `Damping ≈ 67.3498` | ~229 ms |
| **Standard** | Spatial | Default | `0.9` | `700` | `Damping ≈ 47.6235` | ~324 ms |
| **Standard** | Spatial | Slow | `0.9` | `300` | `Damping ≈ 31.1769` | ~496 ms |
| **Both** | Effects | Fast | `1.0` | `3800` | `Damping ≈ 123.2883` | ~149 ms |
| **Both** | Effects | Default | `1.0` | `1600` | `Damping = 80.0` | ~230 ms |
| **Both** | Effects | Slow | `1.0` | `800` | `Damping ≈ 56.5685` | ~326 ms |

> **工程强制规则**：Effects 类颜色/透明度动画启用 `Oklab: true` 与 `NonMotion: true`；Spatial 类动画当 `c.Preferences().ReduceMotion == true` 时立即收敛至目标值（`mygo/ui` 原生尊重 `NonMotion=false` 时的 `ReduceMotion`）。

---

## 3. 形状系统与尺寸分级（Shape & Size Scale）

### 3.1 圆角尺度令牌（Shape Scale）
- `ShapeNone`: `0` DIP
- `ShapeExtraSmall`: `4` DIP
- `ShapeSmall`: `8` DIP
- `ShapeMedium`: `12` DIP
- `ShapeLarge`: `16` DIP
- `ShapeLargeIncreased`: `20` DIP
- `ShapeExtraLarge`: `28` DIP
- `ShapeExtraLargeIncreased`: `32` DIP
- `ShapeExtraExtraLarge`: `48` DIP
- `ShapeFull`: `9999` DIP（或半高，药丸/正圆）

### 3.2 5 档组件尺寸分级（XS / S / M / L / XL）
适用于 Expressive Button、IconButton、ToggleButton、ButtonGroup、SplitButton：

| 尺寸档位 | 容器高度 | 默认圆角（Round / Square） | 按压形变圆角 | 文本字号 | 图标尺寸 | 水平内边距（标准） |
|---|---|---|---|---|---|---|
| **XS (ExtraSmall)** | `32` | Full / `12` | `8` | `14` (LabelLarge) | `20` | `12` |
| **S (Small)** | `40` | Full / `12` | `8` | `14` (LabelLarge) | `20` | `16` |
| **M (Medium)** | `56` | Full / `16` | `12` | `16` (TitleMedium) | `24` | `24` |
| **L (Large)** | `96` | Full / `28` | `16` | `24` (HeadlineSmall) | `32` | `48` |
| **XL (ExtraLarge)** | `136` | Full / `28` | `16` | `32` (HeadlineLarge) | `40` | `64` |

*(注：IconButton 与 SplitButton 依据此尺寸档位按对应宽度、间距与连接内侧圆角微调，详见 `iconbutton.go`、`splitbutton.go` 实现与 GoDoc 说明。)*

### 3.3 35 种 MD3E 几何形状与双射 Morph
`shape` 包须实现完整 35 种内置形状构造器（与 Google `MaterialShapes` 一致）：
1. `Circle`
2. `Square`
3. `Slanted`
4. `Arch`
5. `Fan`
6. `Arrow`
7. `SemiCircle`
8. `Oval`
9. `Pill`
10. `Triangle`
11. `Diamond`
12. `ClamShell`
13. `Pentagon`
14. `Gem`
15. `Sunny`
16. `VerySunny`
17. `Cookie4Sided`
18. `Cookie6Sided`
19. `Cookie7Sided`
20. `Cookie9Sided`
21. `Cookie12Sided`
22. `Ghostish`
23. `Clover4Leaf`
24. `Clover8Leaf`
25. `Burst`
26. `SoftBurst`
27. `Boom`
28. `SoftBoom`
29. `Flower`
30. `Puffy`
31. `PuffyDiamond`
32. `PixelCircle`
33. `PixelTriangle`
34. `Bun`
35. `Heart`

`Morph` 要求：对任意两个归一化的闭合 Cubic Bézier 环按角度/特征对齐重切分（subdivision），确保 `t = 0` 精确等于 `from`、`t = 1` 精确等于 `to`，且所有中间 `t ∈ (0, 1)` 无自交炸裂、平滑闭合。

---

## 4. 排版尺度（Typography Scale — Standard & Emphasized）

默认字体大小/行高/字重/字距（DIP）：

| 角色 | Standard (Size / LineHeight / Weight / Tracking) | Emphasized (Weight 提升 & 表现力细节) |
|---|---|---|
| `DisplayLarge` | `57 / 64 / 400 / -0.25` | Weight `500` |
| `DisplayMedium` | `45 / 52 / 400 / 0` | Weight `500` |
| `DisplaySmall` | `36 / 44 / 400 / 0` | Weight `500` |
| `HeadlineLarge` | `32 / 40 / 400 / 0` | Weight `500` |
| `HeadlineMedium` | `28 / 36 / 400 / 0` | Weight `500` |
| `HeadlineSmall` | `24 / 32 / 400 / 0` | Weight `500` |
| `TitleLarge` | `22 / 28 / 400 / 0` | Weight `500` |
| `TitleMedium` | `16 / 24 / 500 / +0.15` | Weight `600` |
| `TitleSmall` | `14 / 20 / 500 / +0.1` | Weight `600` |
| `BodyLarge` | `16 / 24 / 400 / +0.5` | Weight `500` |
| `BodyMedium` | `14 / 20 / 400 / +0.25` | Weight `500` |
| `BodySmall` | `12 / 16 / 400 / +0.4` | Weight `500` |
| `LabelLarge` | `14 / 20 / 500 / +0.1` | Weight `600` |
| `LabelMedium` | `12 / 16 / 500 / +0.5` | Weight `600` |
| `LabelSmall` | `11 / 16 / 500 / +0.5` | Weight `600` |

---

## 5. MD3E 重点组件专属行为规格

1. **Expressive Button & ToggleButton**：支持 `Filled`、`Tonal`、`Elevated`、`Outlined`、`Text` 五大视觉变体；支持 `Round` 与 `Square` 基础形态；按压时使用 Fast Spatial 弹簧使圆角向按压态收敛（松手回弹）；Toggle 选中态可从 `Round` 变形为 `Square`（或反向）。
2. **Connected / Standard ButtonGroup**：
   - `Standard` 模式下，被按压或选中的按钮沿主轴微微变宽并压缩相邻按钮宽度（弹性权重挤压），同时动态收缩自身圆角。
   - `Connected` 模式下，首项外侧大圆角、末项外侧大圆角、相邻接缝处使用小内圆角（如 `8` 或 `4`），选中项切为全圆角（Full）。
3. **SplitButton**：由主动作段与菜单触发段组成（`SplitButton` 返回 `(group, action, arrow, panel ui.Element)`，与 `ui.SplitButtonParts` 语义对齐）；展开菜单时，触发段图标（Chevron）旋转 180° 且该段四圆角从非对称连接圆角形变为对称圆角/全圆角，关闭时弹簧回位。
4. **FABMenu（取代旧版 Speed Dial）**：必须从标准 FAB（不能从 ExtendedFAB）原地展开 **2–6 个** 大圆角胶囊菜单项（带 `Scrim` 遮罩），原 FAB 变形并切换为高对比关闭按钮（`×` 旋转入场）。
5. **FloatingToolbar**：独立悬浮容器（`56`/`64` 高或垂直列），支持 `Standard` 与 `Vibrant`（`PrimaryContainer` 高亮底色）两种色系，支持紧邻耦合一颗标准 FAB，支持随内容滚动收起/展开选项。
6. **LoadingIndicator（Shape-Morphing）**：
   - 连续在多边形序列（默认 `SoftBurst -> Cookie9Sided -> Pentagon -> Pill -> Sunny -> Cookie4Sided -> Oval` 循环）间以 Expressive Spring 驱动 `shape.Morph` 过渡并伴随柔和旋转。
   - 提供 `Standard`（无背景容器，图形用 `Primary`）与 `Contained`（圆形容器 `PrimaryContainer`，图形用 `OnPrimaryContainer`）两种样式。
   - 离开视口时暂停相位与绘制，重新可见后继续。`Paused` 冻结当前形状。`ReduceMotion` 静态展示首个形状。
7. **Wavy & Linear Progress Indicators（Linear & Circular）**：
   - `LinearProgress` / `LinearWavyProgress`：活跃段在 Wavy 模式下绘制正弦平滑波浪轨迹，确定态（Determinate `0..1`）在接近 0% 或 100% 端点时振幅平滑衰减至 0，未完成轨道留有明确间隙（track gap）及末端停靠点（stop indicator）；支持可选 `ProgressOptions.Buffer`（`0..1`）在主进度与未完成轨道间以 `TrackGap` 间距绘制半透明缓冲段；不确定态（`value < 0`）采用 MD3 双段连续伸缩接力（Primary + Trailing）实现周期无断档流动，并在滚出可视区域（`p.Visible()`）时自动暂停请求动画帧。
   - `CircularProgress` / `CircularWavyProgress`：极坐标正弦波动圆环（$r(\theta) = R + A \sin(k\theta + \phi)$），支持确定态弧长推进（含 `Buffer` 缓冲弧）、不确定态旋转伸缩与视口外零帧暂停。
8. **Expressive List**：支持分节/独立卡片式列表项，首项顶部大圆角、中间项小圆角、末项底部大圆角，选中或按压时独立向大圆角/高亮色平滑过渡。
9. **Expressive Slider & RangeSlider**：轨道高约 `16` DIP（或可配轨高），与拇指（竖直胶囊条 `4 × 44` DIP）之间留有明确的 **Track Gap（`6` DIP）**；离散步进显示停靠点（stop dots）；拖拽或聚焦时弹出上方水滴形（Teardrop）值标签；支持 Centered（从中心向两侧填充）与双滑块 RangeSlider。
10. **Expressive Switch**：轨道 `52 × 32` DIP；未选中拇指 `16` DIP（无图标）或 `24` DIP（带 `×` 图标），选中态弹性放大至 `24` DIP（带 `✓` 图标），按压态瞬间扩大至 `28` DIP，使用 Fast Spatial 弹簧驱动。

---

## 6. 桌面密度系统、表面增强与 Adaptive 断点规格（M9）

### 6.1 三档密度系统（`tokens.Density`）
`DensityComfortable` 为零值默认档位，必须保持上述 M1–M8 全部像素规格 100% 不变；`DensityDesktop` 与 `DensityCompact` 在桌面生产力与高密度信息面板下按以下基线收敛（非法枚举值统一回退至 `DensityComfortable`）：

| 密度档位 | ControlScale / PaddingScale | `ui.Theme.Spacing` | List（1 / 2 / 3 行） | Tabs 高度 | TextField 高度 | NavRail 默认展开宽 | Button 高度（XS / S / M / L / XL） |
|---|---|---|---|---|---|---|---|
| **`DensityComfortable`（默认零值）** | `1.00 / 1.00` | `4` | `56 / 72 / 88` | `48` | `56` | `256` | `32 / 40 / 56 / 96 / 136` |
| **`DensityDesktop`** | `0.85 / 0.85` | `3.5` | `44 / 56 / 72` | `40` | `44` | `192` | `28 / 34 / 48 / 82 / 116` |
| **`DensityCompact`** | `0.75 / 0.70` | `3` | `36 / 48 / 60` | `36` | `38` | `180` | `24 / 30 / 42 / 72 / 102` |

### 6.2 `Surface` 与 `Card` 质感扩展
- **Hairline 微高光描边**：`Hairline: true` 时在未指定 `BorderWidth`（或非 Outlined Card）时绘制 `1` DIP 精致描边；亮色模式默认低透明 `OutlineVariant`（`45%`），暗色模式混入 `15% OnSurface` 后取 `42%` 透明度形成柔和微高光，高对比度模式自动提升为不透明 `Outline` 保证清晰可见；可通过 `HairlineColor` 覆盖。
- **渐变底色与装饰槽位**：支持 `LinearGradient` 与 `RadialGradient`（复用 `ui.Element` 渐变及多边形 `FillPathGradient`/`ClipPath`），支持 Ghost-Draw 安全的 `Backdrop func(p *ui.Painter, r ui.Rect)`（在形状裁切与有效非零尺寸内执行，严禁请求动画帧或读取元素交互态）。
- **Card 圆角档位**：`CardOptions.ShapeScale` 支持选择 MD3E 圆角令牌（如 Bento 卡片使用 `ShapeLargeIncreased` / `ShapeExtraLarge`），零值保持 `ShapeMedium`，正有限值 `Radius` 仍优先覆盖。

### 6.3 Adaptive 断点与桌面外壳
- **WindowSizeClass 断点**：
  - `WindowCompact`：`width < 600` DIP（含 `<= 0`、`NaN`、`-Inf` 安全回退）
  - `WindowMedium`：`600 <= width < 840` DIP
  - `WindowExpanded`：`width >= 840` DIP（含 `+Inf`）
- **`AdaptiveScaffold`**：随断点自动在底部 `NavigationBar`（Compact）、折叠 `NavigationRail`（Medium）与展开 `NavigationRail`（Expanded）间切换，并保持内容槽位位于稳定父级与固定 `Key` 下，确保局部状态不因窗口缩放丢失。
- **内容净宽感知**：`AdaptiveScaffold` 的内容槽位以扣减侧栏实际占用后的可用宽度构建；`BentoGrid` 与 `SupportingPane` 在未显式传入 `WidthOverride` 时通过 `ContentSizeClassOf` 据此解析断点（880 DIP 窗口 + 192 DIP 展开侧栏 → 688 DIP 内容区按 `WindowMedium` 排 2 列且不自动展开辅助栏），避免整窗断点造成的列挤压；`WindowSizeClassOf` 仍恒按整窗宽度判定。
- **`ListDetailPane` / `SupportingPane` / `BentoGrid`**：复用上游 `ui.Split`/`ui.SplitPercent` + `.SplitReset(...)` 与 `ui.Grid`，在 `WindowCompact` 下自动退化为单栏堆叠/受控单页视图，完整支持 RTL 镜像。
- **`Card` 横向约束**：非交互与可点击 Card 均使用 `.AlignItems(ui.Stretch).MinWidth(0)` 并以圆角内裁切约束子内容，窄列中的 `Row(Grow + Fixed)` 不再把卡片撑破边框。
- **`WindowTitleBar`**：基于 `c.TitleBar()` 自动预留 macOS 左交通灯与 Windows/Linux 系统按钮物理安全区，支持 `DragWindow()`、`.MaximizeButton()`（Win11 Snap Layout）与零后端耦合的 `WindowController`（`*mygo.Window` 原生满足该接口）；当保留原生系统按钮（未设 `HideWindowButtons: true`）且绑定 `Window` 时自动通过可选 `SetWindowButtonsTheme` 随 `th.Dark` 同步深浅外观，推荐非 macOS 桌面应用使用 `TitleBarStyle: mygo.TitleBarHidden` + `HideWindowButtons: true` 搭配 `ShowWindowControls: true` 以获得高度对齐的 MD3E 圆角悬停控件；窄屏（`< 520 DIP`）拥挤时自动收敛居中标题避免与右侧控件重叠，且全部按钮禁用 `FocusRing`。

---

## 7. Expressive 异形交互、高频微组件与零膨胀图标子包（M10）

### 7.1 异形按钮与图标徽章（`shapebutton.go`）
- **`ShapeButton` & `MorphIconButton`**：构建在 `ui.ButtonBase` 之上，接入 `Theme.Density.ButtonSpec(size)`，支持 `ShapeButtonFilled`（默认零值）/ `Tonal` / `Outlined` / `Standard` 四种变体与任意 `*shape.RoundedPolygon`。
  - 按压态使用 `motion.FastSpatial`（或自定义 `Spring`）连续 `shape.Morph`；`MorphIconButton` 绑定 `*bool` 并在 `Shape` ↔ `SelectedShape` 间双向 Morph，同时用 `motion.FastEffects` 对 `Icon` ↔ `SelectedIcon` 进行交叉淡入与轻微缩放（`0.75..1.0`），并将 `StateLayer` 与 `Expressive Ripple` 通过 `StateLayerOptions.ClipPath` 严格裁切在当前帧 Morph 轮廓内。
  - 无障碍：`ShapeButton` 使用 `ui.RoleButton`，`MorphIconButton` 使用 `ui.RoleToggleButton` + `Checked(*on)`，二者均强制非空 `Label` 并响应点击、Enter 与 Space，支持 `ReduceMotion` 即时收敛与退场幽灵绘制安全。
- **`IconBadge`**：支持 `Tonal` / `Filled` / `Outlined` / `PrimaryTonal` / `TertiaryTonal` / `ErrorTonal` 及自定义色覆盖，支持 `ShapeScale` / `Radius` 圆角矩形与任意 `*shape.RoundedPolygon` 异形底托。

### 7.2 胶囊标签栏与高频微组件（`pilltabs.go` / `status.go` / `tiles.go` / `feedback_extra.go`）
- **`PillTabs` & `SegmentedTabs`**：基于 `ui.TabsBase` 构建，保留完整 `RoleTabList` / `RoleTab`、Arrow/Home/End 跳过禁用项及 RTL 方向键镜像。胶囊底槽与滑动选中胶囊在 `tabs.List.Draw`（子内容之下）以 `motion.FastSpatial` 弹簧平滑滑动，首帧直接落位于当前选中项而不从 0 飞入；支持 `TabItem.Badge`（同步暴露至 `tab.Description`）与 `TabItem.BadgeDot`。
- **`Tag` & `StatusPill` & `Avatar`**：
  - `SemanticTone`（`Neutral` / `Primary` / `Secondary` / `Tertiary` / `Error` / `Warning` / `Success`），其中 `Warning`（Hue 68°）与 `Success`（Hue 145°）通过 `tokens.HCT` 按亮/暗色与高对比度动态生成谐调容器/前景色。
  - `StatusPill` 的呼吸脉冲圆点（`Pulsing: true`）与 `Skeleton` 的 Tonal 柔和脉冲严格在 `Draw` 内通过 `p.AnimationFrame()` 驱动，结合 `p.Visible()` 视口相交检测保证滚出可视区域后零动画帧，同时遵守 `ReduceMotion` 与 `!el.Valid()` 幽灵安全。
  - `Avatar` 支持中英文首字母提取、`ui.ImageSource`（`ui.Cover`）配合圆形或 `*shape.RoundedPolygon` 裁切、以及 RTL 自适应底端状态圆点（Online/Busy/Away/Offline）。
- **`SwitchCard` & `ActionTile`**：`SwitchCard` 将调用方 `c.Key(...)` 保留在外层卡片、内部 `Switch` 使用独立稳定子 Key，同步整行点击与开关点击而不发生双重翻转；`ActionTile` 基于 `ui.ButtonBase` 提供 `Meta`、`Badge` 与 RTL 自适应 `ChevronRight`/`ChevronLeft`。
- **`EmptyState` & `Banner`**：`EmptyState` 提供插图/异形 `IconBadge` 槽位与主/次 CTA 按钮；`Banner` 支持 `*bool` 可见性、语义色角色（Error/Warning 使用 `ui.RoleAlert`，其余使用 `ui.RoleStatus`）、1–2 个操作按钮与关闭回调。

### 7.3 零膨胀符号子包（`symbols/`）
- 独立公开子包 `github.com/imicola/mygo-md3e/symbols` 仅依赖标准库与 `github.com/egoist/mygo/ui`，导出 **252** 个 24×24 viewBox 圆角矢量符号（分 7 个功能域文件）与 `Custom`/`CustomEvenOdd` 扩展入口。
- 零 `init()`、零全局注册表、零 `sync.OnceValue`；每个符号为独立包级 `*Symbol` 变量，内部使用零值 `sync.Once` 延迟解析 `ui.ParsePath` 并缓存，确保 Go 链接器实现单图标级死代码消除（DCE）。

---

## 8. 大数据虚拟化、层级导航与分页规格（M11）

### 8.1 虚拟化列表与数据表格（`virtuallist.go` / `datatable.go`）
- **`VirtualList`**：构建在 `ui.List` 之上，仅实例化视口附近条目，支持单选（`*int`）与多选（`ui.Selector`）、吸顶分区头、MD3E 分节圆角（首顶大圆角、末底大圆角、选中全大圆角 `SecondaryContainer`）、基于 `ui.ListRow.Hovered`/`Pressed`/`PointerPosition` 的 `DrawOver` 无重建态层与涟漪，以及 LTR/RTL 滚动条安全留白。
- **`DataTable`**：构建在 `ui.Table` 之上，封装 MD3E `Surface` 容器（`Hairline`、默认 `ShapeMedium` 与 tonal `Level1`）、`Theme.Density` 自适应行高（Comfortable `48` / Desktop `38` / Compact `32` DIP）、列拖拽/调宽与双向 `ui.TableLayout` 同步、单/多列排序、`Numeric` 列 `TabularNums` 右对齐、`DataTableCellText` 与 `ApplyDataTableMenuActions` 菜单桥接。

### 8.2 层级导航与分页（`treeview.go` / `breadcrumb.go` / `pagination.go`）
- **`TreeView`**：基于 `ui.TreeBase` 构建，保留完整 `RoleTree` / `RoleTreeItem`（`Level`、`Expandable`、`Expanded`、`Selected`）、单次 Tab 停靠及 Up/Down/Left/Right/Home/End/首字母键入导航。分支披露箭头以 `motion.FastSpatial` 弹簧在 `0° ↔ ±90°` 平滑旋转（RTL 自动镜像箭头与方向键）；点击箭头仅切换展开态且不触发行选中与 `OnClick`；当 `TreeNode.Open == nil` 且存在 `Children` 时通过节点键隔离的 `ui.Local` 持久化展开态；`ShowIndentGuides` 在展开分支下方绘制对齐的 `OutlineVariant` `1` DIP 竖向引导线。
- **`Breadcrumb`**：`ui.RoleNavigation` 地标（默认 `"Breadcrumb"`），祖先层级为 `ui.ButtonBase`，末尾当前项为静态 `RoleText` + `ui.CurrentPage` + `SecondaryContainer` 高亮胶囊；支持 `MaxVisible` 与基于容器 `LayoutBounds` 宽度的响应式自动折叠（中间层级收进 `MoreHoriz` 触发的 `ui.MenuBase` 下拉菜单，保留首项与末尾当前项），自带无限重排保护与 RTL 分隔符镜像。
- **`Pagination`**：1-based（`1..totalPages`）`ui.RoleNavigation` 水平 `FocusGroup` 分页条，安全钳位越界 `*page` 与 `totalPages <= 0`；提供可选 `ShowEdges`（首页/末页）、上一页/下一页与边界禁用逻辑、基于 `SiblingCount` 的窗口省略号折叠及正数 `MaxButtons` 槽位上限（一个槽位显示当前页，两个显示首页和当前页，当前在首页时显示首末页）、`TabularNums` 等宽页码胶囊、`motion.FastSpatial` 按压圆角形变及 RTL 布局/方向键/图标镜像。

---

## 9. 媒体、图像与动态封面取色规格（M12）

### 9.1 图像封面动态取色引擎（`tokens/extract.go`）
- **算法链路（Pure-Go MCU）**：`tokens.SeedFromImage` 与 `tokens.PaletteFromImage` 对 `image.Image` 执行确定性步长下采样（最多 16,384 个样本，按实际行列数调整狭长图像的长轴步长，自动跳过 `alpha < 128` 半透明像素并对预乘 `*image.RGBA` 还原直通通道）、Wu 5-bit `[33][33][33]` 前缀和量化分割、`CIE L*a*b*` 空间 `WSMeans` 质心微调，以及基于 `360°` 色相直方图（`±15°` 激励比例）与色度加权的 Material `Score` 排序（含渐进最小色相间隔去重与 `#6750A4` 等可配置回退色）。
- **零重算防抖原则**：所有播放器仅在封面图像引用变化时触发提取并将结果记忆在 `ui.Local`，杜绝每帧遍历像素。

### 9.2 根包局部图像 `md3e.Image`（`image.go`）
- **多形态与适应度**：支持 `*ui.Bitmap`、`*ui.SVG` 与通用 `ui.ImageSource`，提供 `Fit`（默认 `Contain`）、`ObjectPosition`（默认 `[0.5, 0.5]`）与 `Grayscale`；支持 `ShapeScale` / `Radius` / `Radii` 四角独立圆角或任意 `*shape.RoundedPolygon` 几何裁切（`shape.ClipShape` 或复用路径缓冲的 `shape.ClipShapeTo`）。
- **状态迁移与幽灵绘制**：`Loading` 复用适配形变的 Tonal `Skeleton`；`Failed` / `Error` 渲染图标容器与可选重试按钮；异步 `Loading/nil -> src` 过渡使用 `motion.DefaultEffects` 从 `0 -> 1` 平滑淡入并响应 `ReduceMotion` / `NoFade`；支持可选 `MatchKey` 共享元素过渡；严守根包零引入 `media/`、`ui/netimage` 与 `ui/video`。

### 9.3 网络图、模态灯箱与 Expressive 音视频播放器（`media/`）
- **`media.NetImage`**：在 `github.com/egoist/mygo/ui/netimage` 上叠加 MD3E 骨架屏占位、网络失败重试（点击通过 `c.Services()` + `ForgetImage` 清除失败退避后立即重新拉取）与空白 URL 安全空占位，并导出 `PreloadImages`、`ForgetImage`、`ClearImageCache` 与 `SetImageHTTPClient`。
- **`media.ImageLightbox`**：基于全屏 `ui.Overlay` + `Modal()` 的照片预览器（含退场动画隔离与关闭归还焦点）。支持工具栏 / 滚轮 / 双指捏合 / 双击 / `=` `-` `0` 键 `1x..5x` 缩放（默认），支持按放大倍比与视口尺寸钳位的拖拽与触控板平移、RTL 自动镜像的上/下页按钮与左右方向键切换、顶部计数栏、底部标题/副标题浮层、`MatchKey` 共享元素进场及 `Escape`/Scrim 外部点击关闭。
- **`media.VideoPlayer`**：封装 `ui.Video` / `*ui.VideoPlayer` 与 `OpenVideo`/`OpenVideoURL`/`OpenVideoData` 辅助构造器。提供基于虚拟时钟（`c.Now()` 与可见时的 `p.After`）在播放闲置 `AutoHideDelay`（默认 `3s`）后平滑收起、鼠标移入/按键/操作即刻唤起的底部渐变与半透明控制条；集成 `MorphIconButton`（`Circle ↔ Square`）播放切换、共享 `mediaScrubber`（播放态 `LinearWavy` 波浪与缓冲进度段 + 拖拽/聚焦时显示水滴 `TabularNums` 时间气泡）、`TabularNums` 时长文本、静音开关与音量条、倍速 `MenuButton`、`Fullscreen` 钩子、居中缓冲 `LoadingIndicator`、错误重试覆盖层及全套键盘快捷键（`Space`/`K`、`M`、`F`、RTL 感知 `Left`/`Right`）。
- **`media.AudioPlayerCard` & `media.MiniPlayer`**：支持 `TrackInfo` 封面显示与可选异形裁切（`ArtworkShape`），并在 `DynamicColor: true` 或 `SeedColor` 开启时记忆提取主题并包裹于 `md3e.AnimatedThemeScope` 实现跨曲目 Oklab 平滑换肤。`AudioPlayerCard` 提供展开封面信息、波浪拖拽轨、时间显示与 `FocusGroup` 传输按钮栏；`MiniPlayer` 随 `Theme.Density` 自适应高度（`64`/`56`/`52` DIP）并在底部边沿使用仅播放且未启用 `ReduceMotion` 时波动的 `LinearWavyProgress`（暂停与 `ReduceMotion` 下切换静止 `LinearProgress`，保证 `FramePending() == false`）。
- **DCE 硬边界验证**：通过独立编译 `media/dce_test.go` 验证仅引用 `md3e.Button`+`md3e.Image` 时零链接 `media/`、`ui/netimage`、`ui/video` 与 `symbols/`，且仅引用 `media.NetImage` 时零链接 `ui/video` 与 `tokens.SeedFromImage`。

---

## 10. 实时与统计图表子包规格（M14：`charts/`）

### 10.1 HCT 谐调多序列色板（`charts/palette.go`）
- **`SeriesColor` / `SeriesPalette`**：槽位 `0..2` 优先锚定当前主题 `Primary`、`Tertiary`、`Secondary`（并在 `TonalSpot` 等 `Secondary` 与 `Primary` 同色相方案中自动检测色相夹角 `< 28°` 切入感知分离偏转角，杜绝多序列撞色），更高槽位基于主色相按预设感知分离角与黄金角续延派生；`Monochrome` 单色主题保持中性灰度色阶区分而不注入人工高饱和色。
- **对比度硬底线**：每个序列色相对卡片表面（`Level1`）均通过 `tokens.ContrastRatio` 校验并在不足时沿 HCT Tone 轴微调至达标（标准对比度 `>= 3.0:1`、中等 `>= 3.5:1`、高对比度 `>= 4.5:1`），`SeriesColor` 单点求值保持零堆分配。

### 10.2 实时环形缓冲与零分配面积微图（`ringbuffer.go` / `sparkline.go`）
- **`RingBuffer`**：O(1) 零分配定长环形缓冲，保留 `NaN` 断点，`MinMax` 自动跳过非有限值，`CopyTo` 复用目标切片达成零分配。
- **`AreaSparkline`**：默认 `CurveSmooth`（Steffen 单调三次平滑样条，兼容 `CurveLinear` 与 `CurveStep*`），通过 `ui.Local` 持久化 `sparklineScratch` 复用 `linePath`/`areaPath`/`clipPath` 与坐标/斜率/降采样切片（高密采样超过 `2×pixelWidth` 时复用 `chart.MinMax`），核心路径构造实测 `0 B/op`、`0 allocs/op`；绘制三段垂直渐变面积淡出、柔和曲线外发光底描边、可选带表面环与光晕的末端实时点，并支持 `BottomRadius`/`Radii` 独立四角裁切以无缝贴合卡片底边。

### 10.3 仪表盘占比条与指标卡片（`stacked.go` / `metriccard.go`）
- **`StackedProgress`**：外端胶囊圆角 + 内部 `3 DIP` 间隙与内圆角 + 未占用余量轨 + 柔和顶部微高光，支持 `Theme.Density` 自适应高度（`12`/`10`/`8` DIP）、`DefaultSpatial` 弹簧宽度过渡（`ReduceMotion` 零帧收敛）、`ui.RightToLeft` 镜像排布、悬停单段微高亮与即时浮动占比提示，以及自动换行 `TabularNums` 图例与 `RoleMeter` 无障碍摘要。
- **`MetricCard`**：整合可选 `IconBadge`、标题/副标、语义 `Tag` 或呼吸 `StatusPill`、纯绘制幽灵安全的局部主题径向氛围底光、Hairline 微高光描边、密度自适应 `TabularNums` 大号读数/单位/峰值，以及零内边距满宽贴合底部圆角的 `AreaSparkline` 或 `LinearWavyProgress`/`LinearProgress` 槽位。

### 10.4 通用统计图表与 DCE 边界（`tooltip.go` / `linechart.go` / `barchart.go` / `donutchart.go` / `dce_test.go`）
- **`LineChart` / `BarChart` / `DonutChart`**：薄封装上游 `github.com/egoist/mygo/ui/chart`（`Plot`/`NewLinear`/`NewBand`/`Stack`/`Pie`/`Sector`/`SliceAt`/`Polar`/`Select`/`Anchor`），统一具备：
  - 指针穿透的 MD3E `RichTooltip` 风格浮动读数卡（`SurfaceContainerHigh` + `Level2` 阴影 + `OutlineVariant` 描边 + `TabularNums` + 边缘自动翻转防裁切）；
  - 可点击聚焦单序列并柔暗兄弟序列的交互式图例；
  - `DefaultSpatial` 弹簧入场与 `DonutChart` 归一化 `FastSpatial` 激活扇区外弹（`+6 DIP` + 外缘光晕 + 动态中心指标读数）；
  - 完整键盘导航（Tab/Arrow/Home/End/Escape + `c.Announce`）、`DonutChart` RTL 方向键镜像、`ReduceMotion` 首帧零多余帧与退场幽灵绘制安全。
- **DCE 硬边界验证**：通过离线 `charts/dce_test.go` 验证仅引用根包 `md3e.Button` 时零链接 `charts/` 与 `ui/chart`，且仅引用 `charts.AreaSparkline` + `charts.RingBuffer` 时零链接 `charts.BarChart`、`charts.DonutChart`、`symbols/` 与 `media/`。

---

## 11. 文档、Markdown 与代码视图子包规格（M13：`docview/` & `codeview/`）

### 11.1 `codeview` 编辑器与 Diff 主题桥（`codeview/theme.go`）
- **`ThemeFromMD3E`**：将 `*md3e.Theme` 映射到 `*editor.Theme`，包括 `Surface` / `SurfaceContainerLowest` 背景、当前行 `SurfaceContainerHighest` 高亮、选区半透明 `Primary`、匹配括号与搜索高亮、行号槽与 `OutlineVariant` 细分割线、`Primary` 光标、Diff 增删改三态表面底色（兼容 Light/Dark/HighContrast 对比度收敛），以及 9 类语法 Token 调色。
- **`FontFromMD3E` / `ScrollbarFromMD3E` / `ApplyToEditor` / `ApplyToDiff` / `ConfigureEditor`**：提供等宽排版字阶映射、`md3e` 胶囊滚动条风格以及编辑器/Diff 状态一步配置与实时上下文字体/滚动条同步。

### 11.2 `docview.Markdown` 与流式输出（`docview/markdown.go`）
- **`MarkdownTheme` / `MarkdownThemeFor`**：基于 `th.Type`、`th.Colors` 与 `th.Density` 生成 `*markdown.Theme`，统一映射 `H1..H6`（H1–H3 使用 `Brand` 字体族）、正文行高、行内代码微胶囊、引用左/Start 装饰竖线、`SurfaceContainer` 表格斑马条纹、圆角、链接与文本选区颜色；默认复用 `codeview.ThemeFromMD3E(th)` 着色源码片段。
- **`Markdown` / `MarkdownStream` / `Revealed`**：`Markdown` 自动将围栏代码块委托给 `docview.CodeBlock`（流式未闭合块自动禁用复制按钮并等待输出完结），将 GitHub Alert 引用映射为 `docview.Callout`，同时保持 `ui/markdown` 的 `ShareSelection` 与 `Copy as Markdown` 完整可用；`MarkdownStream` 结合 `Theme.ReduceMotion || c.Preferences().ReduceMotion` 控制渐显动画并确保结束后无残留动画帧；提供 `Anchor` / `AnchorVersion` 与 `*docview.State` 联动。

### 11.3 文档增强块与滚动联动目录（`codeblock.go` / `callout.go` / `state.go` / `outline.go`）
- **`docview.CodeBlock`**：`SurfaceContainerHigh` + `ShapeMedium`（可调）+ 1 DIP `Hairline` 描边卡片，含语言徽章与可选标题栏、可选 `TabularNums` 不可选中行号槽、强制 `ui.LeftToRight` 代码排版方向、`ui/editor` 语法高亮着色与横向滚动条，以及右上角一键复制按钮（支持点击后 2 秒勾选反馈、同步写入剪贴板与可选 `md3e.SnackbarState` 提示、退场幽灵安全与禁用状态）。
- **`docview.Callout`**：`CalloutNote`、`CalloutTip`、`CalloutImportant`、`CalloutWarning`、`CalloutCaution` 五类语义块（配合 `CalloutKindFromAlert`），分别映射至 `PrimaryContainer`/`SuccessContainer`/`TertiaryContainer`/`WarningContainer`/`ErrorContainer`（均由底层色彩回落保障可用性），提供 Start 侧 3 DIP 强调竖条（支持 `ui.RightToLeft` 右缘镜像）、`RoleRegion`/`RoleAlert` 无障碍语义、Markdown 原生子内容或纯文本降级渲染。
- **`docview.State` / `ExtractOutline` / `Outline`**：`ExtractOutline` 解析文档标题生成去重小写 `Slug`、`Level` 与块索引；`Markdown` 构建期通过 `LayoutBounds()` 测量标题相对垂直偏移且不干扰 `ui.Scroll` 视口裁剪；`Outline` 在 `RoleNavigation` 内提供层级缩进、活跃态 `SecondaryContainer` 胶囊背景 + `FastSpatial` 弹簧过渡 + Primary Start 侧 3 DIP 指示条（RTL 自动镜像）、点击平滑滚动（无 `ui.ScrollState` 时平滑回退至 `ui/markdown` 锚点）、`FocusGroup(ui.Vertical)` 方向键导航、单独传入 `Items`/`Active`/`OnSelect` 与与 `md3e.SupportingPane` 协同布局。
- **DCE 硬边界验证**：通过离线 `codeview/dce_test.go` 与 `docview/dce_test.go` 验证仅引用 `md3e.Button` 时零链接 `codeview/`、`docview/` 与 `ui/markdown`/`ui/editor`；仅调用 `codeview.ThemeFromMD3E` 时零链接 `editor.View`/`editor.DiffView`；仅引用 `docview.Outline` + `docview.State` 时零链接 `editor.View`/`editor.DiffView`、`charts/` 与 `media/`。


