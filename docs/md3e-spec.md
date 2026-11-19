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

*(注：IconButton 与 SplitButton 依据此尺寸档位按对应宽度、间距与连接内侧圆角微调，详见各组件任务点。)*

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
3. **SplitButton**：由主动作段与菜单触发段组成（`ui.SplitButtonParts` 语义对齐）；展开菜单时，触发段图标（Chevron）旋转 180° 且该段四圆角从非对称连接圆角形变为对称圆角/全圆角，关闭时弹簧回位。
4. **FABMenu（取代旧版 Speed Dial）**：必须从标准 FAB（不能从 ExtendedFAB）原地展开 **2–6 个** 大圆角胶囊菜单项（带 `Scrim` 遮罩），原 FAB 变形并切换为高对比关闭按钮（`×` 旋转入场）。
5. **FloatingToolbar**：独立悬浮容器（`56`/`64` 高或垂直列），支持 `Standard` 与 `Vibrant`（`PrimaryContainer` 高亮底色）两种色系，支持紧邻耦合一颗标准 FAB，支持随内容滚动收起/展开选项。
6. **LoadingIndicator（Shape-Morphing）**：
   - 连续在多边形序列（默认 `SoftBurst -> Cookie9Sided -> Pentagon -> Pill -> Sunny -> Cookie4Sided -> Oval` 循环）间以 Expressive Spring 驱动 `shape.Morph` 过渡并伴随柔和旋转。
   - 提供 `Standard`（无背景容器，图形用 `Primary`）与 `Contained`（圆形容器 `PrimaryContainer`，图形用 `OnPrimaryContainer`）两种样式。
   - `ReduceMotion` 时静止展示首帧特征形状，不请求动画帧。
7. **Wavy Progress Indicators（Linear & Circular）**：
   - `LinearWavyProgress`：活跃段绘制正弦平滑波浪轨迹，确定态（Determinate `0..1`）在接近 0% 或 100% 端点时振幅平滑衰减至 0，未完成轨道留有明确间隙（track gap）及末端停靠点（stop indicator）；不确定态（Indeterminate）持续相移流动。
   - `CircularWavyProgress`：极坐标正弦波动圆环（$r(\theta) = R + A \sin(k\theta + \phi)$），支持确定态弧长推进与不确定态旋转伸缩。
8. **Expressive List**：支持分节/独立卡片式列表项，首项顶部大圆角、中间项小圆角、末项底部大圆角，选中或按压时独立向大圆角/高亮色平滑过渡。
9. **Expressive Slider & RangeSlider**：轨道高约 `16` DIP（或可配轨高），与拇指（竖直胶囊条 `4 × 44` DIP）之间留有明确的 **Track Gap（`6` DIP）**；离散步进显示停靠点（stop dots）；拖拽或聚焦时弹出上方水滴形（Teardrop）值标签；支持 Centered（从中心向两侧填充）与双滑块 RangeSlider。
10. **Expressive Switch**：轨道 `52 × 32` DIP；未选中拇指 `16` DIP（无图标）或 `24` DIP（带 `×` 图标），选中态弹性放大至 `24` DIP（带 `✓` 图标），按压态瞬间扩大至 `28` DIP，使用 Fast Spatial 弹簧驱动。
