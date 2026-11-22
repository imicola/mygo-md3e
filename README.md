# mygo-md3e

English | [简体中文](README.zh-CN.md)

**mygo-md3e** is a pure-Go **Material Design 3 Expressive (MD3E)** native component library built on top of [MyGo](https://github.com/egoist/mygo)'s GPU-rendered `ui` toolkit (`github.com/egoist/mygo/ui`). Every component is drawn directly in Go with zero HTML, zero WebView, and zero cgo, running from a single codebase on desktops (macOS, Windows, Linux) and in browsers (`GOOS=js GOARCH=wasm`).

## Highlights

- **Pure Go (`CGO_ENABLED=0`)**: Compiles and passes tests without cgo, with verified cross-compilation for `linux/amd64`, `darwin/arm64`, and `js/wasm`.
- **Full MD3E Design System (`tokens`)**: HCT / CAM16 color space, `TonalPalette`, 7 `DynamicScheme` variants (`TonalSpot`, `Expressive`, `Vibrant`, `Neutral`, `Monochrome`, `Fidelity`, `Content`), 3 contrast tiers, 45+ semantic color roles, 10 shape scales, XS–XL size tiers, 15 standard + 15 emphasized typography styles, and 6 elevation levels.
- **Physics-Driven Spring Motion (`motion`)**: 12 Expressive and Standard spring presets (Spatial & Effects) converting MD3E damping ratio $\zeta$ and stiffness $k$ to MyGo's viscous damping coefficient $c = 2\zeta\sqrt{k}$ (`Mass = 1`), with Oklab interpolation for effect tokens.
- **35 Expressive Shapes & Bidirectional Morphing (`shape`)**: Built-in `RoundedPolygon` primitives (`Circle` through `Heart`) and bijective cubic Bézier morphing rendered and clipped via `ui.Path`.
- **Zero-Bloat Vector Symbol Library (`symbols`)**: 252 rounded 24×24 vector symbols across 7 functional domains with zero `init()` or global registries for verified per-icon dead-code elimination; `symbols.<Name>.Draw` plugs directly into any `md3e.IconDraw` slot.
- **Rich Component Suite (`md3e`)**: Action buttons, shape-morphing buttons (`ShapeButton` / `MorphIconButton`), split buttons, button groups, FABs & FAB menus, floating toolbars & app bars, selection controls, sliders, chips, text fields, shape-morphing loading indicators, wavy progress bars, tooltips, snackbars, cards, `SwitchCard`/`ActionTile`, segmented & virtualized lists (`VirtualList`, `DataTable`), navigation bar/rail/drawer, `TreeView`, responsive `Breadcrumb`, `Pagination`, underline & pill/segmented tabs, `Tag`/`StatusPill`/`Avatar`/`Skeleton`/`EmptyState`/`Banner`, carousels, dialogs, bottom sheets, menus, date/time pickers, desktop `WindowTitleBar`, and adaptive layouts.
- **Zero-Bloat Dead-Code Elimination**: No package-level `init()` component or icon registries—applications only link the components and built-in vector icons they actually reference.
- **Accessible, RTL-Ready, Headless-Testable**: Built over `ui.*Base` primitives with keyboard navigation, `ReduceMotion` frame-settling, right-to-left mirroring, and deterministic `ui.NewTester` + `ui/golden` verification.

---

## Requirements & Installation

- **Go 1.27.3+** (aligned with `github.com/egoist/mygo`).
- Depends directly on upstream `github.com/egoist/mygo` (`v0.10.1-0.20261119113012-61ea5567ab14`), with no local checkout or `replace` directive required.

Install into an existing Go module:

```sh
go get github.com/imicola/mygo-md3e@latest
```

---

## Quick Start

### Run the Interactive Gallery

`examples/gallery` is an interactive multi-page `ui.Router` showcase covering every package and component, with a collapsible `NavigationRail` and live controls for Seed Color, Variant, Contrast, Light/Dark, Motion Scheme, `ReduceMotion`, and RTL:

```sh
CGO_ENABLED=0 go run ./examples/gallery
```

`examples/desktop-demo` is PulseDesk, a compact desktop workbench that pairs the M9 shell (`WindowTitleBar`, density profiles, adaptive list/detail and supporting panes, Bento grid) with the M10 expressive layer (`ShapeButton`/`MorphIconButton`, `PillTabs`/`SegmentedTabs`, tags, status pills, avatars, skeletons, empty states, banners) and the zero-bloat `symbols` pack:

```sh
CGO_ENABLED=0 go run ./examples/desktop-demo
```

The earlier M1–M2 interactive preview (`tokens`, `motion`, `shape`) is also available as a standalone module under `examples/m2-demo`:

```sh
cd examples/m2-demo
CGO_ENABLED=0 go run .
```

### Minimal Hello MD3E

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

Key conventions:

1. Every component takes `c *ui.Context` as its first parameter and binds state via pointers (`*bool`, `*float64`, `*int`, `*string`, `*time.Time`), applying user input during construction.
2. Call `c.Key(...)` before constructing stateful controls (`TextField`, `Switch`, `Slider`, etc.), and call `e.Key(...)` on containers before reading element state or calling `ui.Local`.
3. `md3e.ThemeScope` bridges `ui.Themed`, so overlays opened via `ui.Overlay` (`AlertDialog`, `BottomSheet`, `DropdownMenu`, `DatePicker`, `SnackbarHost`, etc.) inherit the active theme.
4. `ThemeScope` defaults to content height. When used as the full-window root wrapper around `.Fill()` or `.Grow(1)` scroll containers, chain `.Fill()` on `ThemeScope` (as shown above); omit `.Fill()` for nested cards or partial sections.

---

## Architecture & Package Layout

```text
.
├── tokens/            # HCT/CAM16 color science, TonalPalette, DynamicScheme, shape/type/elevation/size tokens
├── motion/            # Expressive & Standard spring physics (ζ -> c conversion, 12 motion tokens)
├── shape/             # RoundedPolygon primitives, 35 built-in MD3E shapes, bijective cubic Bézier Morph
├── symbols/           # 252 zero-bloat rounded 24x24 vector symbols (per-icon DCE, Draw matches IconDraw)
├── internal/mdicon/   # Zero-registry 24x24 vector icons used by library chrome
├── theme.go           # Theme (with Density & MonoFont), ThemeFromSeed, DefaultTheme, ThemeScope, CurrentTheme
├── surface.go         # Surface container, tonal elevation, shadows, Hairline, gradients, Backdrop slot
├── typography.go      # Text, TextWith, TabularNums, TruncateMode, ApplyTypeStyle, TypeStyleOf
├── ripple.go          # AttachStateLayer, AttachRipple (DrawOver state tracking + pointer-origin ripples)
├── button.go …        # Actions & bars: Button, IconButton, ButtonGroup, SplitButton, FAB, FABMenu, FloatingToolbar, AppBar
├── switch.go …        # Selection & inputs: Switch, Checkbox, Radio, Slider, RangeSlider, Chips, TextField, TextArea
├── loading.go …       # Feedback & progress: LoadingIndicator, Linear/Circular/Wavy Progress, Badge, Tooltip, Snackbar
├── card.go …          # Containers, navigation & overlays: Card, Divider, List, NavBar/Rail/Drawer, Tabs, Carousel, Dialog, Sheet, Menu
├── datepicker.go      # DatePicker (Docked / Modal) & DateRangePicker
├── timepicker.go      # TimePicker (Dial / Input, 12h / 24h)
├── shell.go           # Desktop WindowTitleBar & WindowController
├── adaptive.go        # WindowSizeClass, AdaptiveScaffold, ListDetailPane, SupportingPane, BentoGrid/BentoSpan
├── shapebutton.go     # ShapeButton, MorphIconButton, IconBadge
├── pilltabs.go        # PillTabs & SegmentedTabs
├── status.go          # SemanticTone, Tag, StatusPill, Avatar
├── tiles.go           # SwitchCard & ActionTile
├── feedback_extra.go  # Skeleton, EmptyState, Banner
├── virtuallist.go     # VirtualList (ui.List virtualization + DrawOver state layers & ripples)
├── datatable.go       # DataTable, DataTableCellText, ApplyDataTableMenuActions (ui.Table wrapper)
├── treeview.go        # TreeView (ui.TreeBase wrapper + indent guides & spring disclosure)
├── breadcrumb.go      # Breadcrumb (responsive path bar + MoreHoriz overflow dropdown menu)
├── pagination.go      # Pagination (1-based pill bar + TabularNums & window ellipsis folding)
├── image.go           # Local/SVG Image with shape clipping, smooth fade-in, Skeleton & retry slots
├── media/             # Optional extension subpackage: NetImage, ImageLightbox, VideoPlayer, AudioPlayerCard, MiniPlayer
├── charts/            # Optional extension subpackage: RingBuffer, AreaSparkline, StackedProgress, MetricCard, LineChart, BarChart, DonutChart
├── docs/              # Architecture guide & MD3E specification baseline
└── examples/
    ├── gallery/       # Full interactive showcase application (part of root module)
    ├── desktop-demo/  # PulseDesk compact desktop workstation example
    └── m2-demo/       # Standalone M1–M2 tokens/motion/shape preview module
```

Dependencies are strictly acyclic: `tokens` -> `motion` -> `shape` -> root `md3e` -> opt-in `media` / `charts`, and `symbols` depends only on `github.com/egoist/mygo/ui` + stdlib (`tokens`/`motion`/`shape`/`symbols` never import `md3e`, and root `md3e` never imports `media/`, `charts/`, `ui/netimage`, `ui/video`, or `ui/chart`).

---

## Component & API Catalog

### Design Tokens (`tokens`)

| Area | Entry Points |
| --- | --- |
| Color Space & Contrast | `tokens.HCT`, `tokens.HCTFromColor`, `tokens.CAM16`, `tokens.ContrastRatio`, `tokens.ContrastTone` |
| Tonal & Dynamic Schemes | `tokens.TonalPalette`, `tokens.NewDynamicScheme` (`TonalSpot`, `Expressive`, `Vibrant`, `Neutral`, `Monochrome`, `Fidelity`, `Content`) |
| Contrast Levels | `tokens.ContrastStandard`, `tokens.ContrastMedium`, `tokens.ContrastHigh` |
| Shape, Size, Density, Type & Elevation | `tokens.ShapeScale` (`ShapeNone`..`ShapeFull`), `tokens.SizeTier` (`SizeXS`..`SizeXL`), `tokens.Density` (`DensityComfortable`, `DensityDesktop`, `DensityCompact`), `tokens.DensitySpec`, `tokens.TypeRole`, `tokens.NewTypographyScale`, `tokens.ElevationLevel` (`Level0`..`Level5`) |

### Motion & Shapes (`motion`, `shape`)

| Area | Entry Points |
| --- | --- |
| Spring Schemes & Tokens | `motion.Expressive`, `motion.Standard`, `motion.FastSpatial`..`motion.SlowEffects` |
| Resolution Helpers | `motion.SpecOf`, `motion.Spring`, `motion.Resolve`, `motion.Snap` |
| 35 Expressive Shapes | `shape.All()`, `shape.ByName`, `shape.Circle`..`shape.Heart`, `shape.RegularPolygon`, `shape.Star` |
| Bidirectional Morphing | `shape.NewMorph`, `Morph.AsPath`, `shape.DrawMorph`, `shape.ClipShape` |

### Core Primitives (`md3e`)

`Theme`, `ThemeOptions`, `ThemeFromSeed`, `DefaultTheme`, `ThemeScope`, `CurrentTheme`, `DefaultSeedColor`, `ShapeScale`, `DefaultShapeScale`, `Surface`, `SurfaceOptions`, `SurfaceColorForElevation`, `ApplyTonalElevation`, `ApplyElevationShadows`, `Text`, `TextWith`, `TextOptions`, `TruncateMode`, `TabularNums`, `ApplyTypeStyle`, `TypeStyleOf`, `AttachStateLayer`, `AttachRipple`, `IconDraw`.

### Actions & Bars

| Component | Entry Points |
| --- | --- |
| Button (5 variants × 5 sizes × Round/Square) | `Button` |
| Expressive Shape Buttons & Icon Badge | `ShapeButton`, `MorphIconButton`, `IconBadge` |
| IconButton & Morphing ToggleIconButton | `IconButton`, `ToggleIconButton` |
| ButtonGroup (Standard / Connected / Multi / Actions) | `ButtonGroup`, `ButtonGroupMulti`, `ButtonGroupActions` |
| SplitButton | `SplitButton` |
| FAB, ExtendedFAB & FABMenu (2–6 items) | `FAB`, `ExtendedFAB`, `FABMenu` |
| FloatingToolbar (Horizontal/Vertical, coupled FAB) | `FloatingToolbar` |
| TopAppBar & SearchAppBar | `TopAppBar`, `SearchAppBar` |

### Selection & Inputs

| Component | Entry Points |
| --- | --- |
| Switch | `Switch` |
| Checkbox (Off / On / Mixed) | `Checkbox`, `CheckboxState` |
| Radio & RadioGroup | `Radio`, `RadioGroup` |
| Slider & RangeSlider (steps, stops, centered, teardrop indicator) | `Slider`, `RangeSlider` |
| Chips | `AssistChip`, `SuggestionChip`, `FilterChip`, `InputChip`, `ChipGroup` |
| TextField & TextArea (Filled / Outlined) | `TextField`, `TextArea` |

### Progress & Feedback

| Component | Entry Points |
| --- | --- |
| Shape-Morphing LoadingIndicator (Standard / Contained) | `LoadingIndicator`, `DefaultLoadingPolygons` |
| Linear / LinearWavy / Circular / CircularWavy Progress | `LinearProgress`, `LinearWavyProgress`, `CircularProgress`, `CircularWavyProgress` |
| Badge | `Badge`, `SmallBadge`, `BadgedBox` |
| Tags, Status & Avatar | `Tag`, `StatusPill`, `Avatar`, `SemanticTone` |
| Skeleton, EmptyState & Banner | `Skeleton`, `EmptyState`, `Banner` |
| Tooltip | `PlainTooltip`, `RichTooltip` |
| Snackbar | `SnackbarHost`, `SnackbarState` |

### Containers, Navigation, Overlays & Pickers

| Component | Entry Points |
| --- | --- |
| Local & Vector Image (`md3e`) | `Image`, `ImageOptions` (`Fit`, `ShapeScale`/`Radii`/`Shape`, `Skeleton` loading, `OnRetry`, `FadeIn`, `MatchKey`) |
| Cover Seed Extraction (`tokens`) | `tokens.SeedFromImage`, `tokens.PaletteFromImage`, `tokens.ExtractOptions` |
| Network Image & Lightbox (`media`) | `media.NetImage`, `media.ImageLightbox`, `media.PreloadImages`, `media.ForgetImage`, `media.ClearImageCache`, `media.SetImageHTTPClient` |
| Video & Audio Players (`media`) | `media.VideoPlayer`, `media.OpenVideo`, `media.OpenVideoURL`, `media.OpenVideoData`, `media.AudioPlayerCard`, `media.MiniPlayer`, `media.TrackInfo` |
| Dashboard Micro-Charts (`charts`) | `charts.RingBuffer`, `charts.AreaSparkline`, `charts.StackedProgress`, `charts.MetricCard`, `charts.SeriesColor`, `charts.SeriesPalette` |
| Statistical Charts (`charts`) | `charts.LineChart`, `charts.BarChart` (`BarGrouped`/`BarStacked`), `charts.DonutChart` (`Donut`/`Pie`), `charts.TooltipRow` |
| Cards & Divider | `ElevatedCard`, `FilledCard`, `OutlinedCard`, `Card`, `Divider` |
| Quick Toggle & Action Tiles | `SwitchCard`, `ActionTile` |
| Expressive Segmented List & Virtualized Data | `List`, `ListItem`, `VirtualList`, `DataTable`, `DataTableCellText`, `ApplyDataTableMenuActions` |
| Navigation Bar / Rail / Drawer | `NavigationBar`, `NavigationRail` (`NavigationRailIndicatorStyle`), `NavigationDrawer` |
| Hierarchical Tree, Breadcrumb & Pagination | `TreeView` (`TreeNode`, `TreeViewOptions`), `Breadcrumb` (`BreadcrumbItem`, `BreadcrumbOptions`), `Pagination` (`PaginationOptions`) |
| Page Transition Container | `PageTransition`, `AnimatedContent` (`TransitionFadeThrough` / `TransitionSharedAxisY` / `TransitionSharedAxisX` / `TransitionNone`) |
| Sliding-Indicator & Pill Tabs | `PrimaryTabs`, `SecondaryTabs`, `PillTabs`, `SegmentedTabs` |
| Carousel (Hero / MultiBrowse / Uncontained) | `Carousel` |
| Dialogs & BottomSheet | `AlertDialog`, `FullScreenDialog`, `BottomSheet` |
| Menus | `DropdownMenu`, `MenuButton`, `ContextMenu` |
| Date & Time Pickers | `DatePicker`, `DateRangePicker`, `TimePicker` |
| Desktop Shell & Adaptive Layouts | `WindowTitleBar`, `WindowController`, `WindowSizeClass`, `WindowSizeClassForWidth`, `WindowSizeClassOf`, `ContentSizeClassOf`, `AdaptiveScaffold`, `ListDetailPane`, `SupportingPane`, `BentoGrid`, `BentoSpan` |
| Zero-Bloat Vector Symbols (`symbols`) | 252 rounded 24×24 icons (`symbols.Home`, `symbols.PlayArrow`, `symbols.Pause`, `symbols.Settings`, `symbols.Terminal`, …), `symbols.Custom`, `symbols.CustomEvenOdd` |

### Using Zero-Bloat `symbols` & Expressive Shape Controls

```go
import (
	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// Every symbol's Draw method matches md3e.IconDraw with per-icon DCE:
md3e.MorphIconButton(c.Key("play"), &playing, "Play or pause", symbols.PlayArrow.Draw,
	md3e.MorphIconButtonOptions{
		Size:          tokens.SizeM,
		Shape:         shape.Cookie9Sided(),
		SelectedShape: shape.Circle(),
		SelectedIcon:  symbols.Pause.Draw,
	})

md3e.SwitchCard(c.Key("proxy"), "System Proxy", &proxyOn, md3e.SwitchCardOptions{
	Subtitle:    "Route desktop traffic automatically",
	LeadingIcon: symbols.Public.Draw,
	Hairline:    true,
})

md3e.Breadcrumb(c.Key("path"), []md3e.BreadcrumbItem{
	{Label: "Workspace", Icon: symbols.Home.Draw, OnClick: goHome},
	{Label: "Docs", Icon: symbols.Folder.Draw, OnClick: goDocs},
	{Label: "Navigation", Icon: symbols.Description.Draw},
})

md3e.TreeView(c.Key("files"), []md3e.TreeNode{
	{Key: "docs", Label: "docs", Icon: symbols.Folder.Draw, Children: []md3e.TreeNode{
		{Key: "docs/guide.md", Label: "guide.md", Icon: symbols.Description.Draw},
	}},
}, md3e.TreeViewOptions{Selected: &selectedKey, ShowIndentGuides: true})

md3e.Pagination(c.Key("pager"), &page, 20, md3e.PaginationOptions{ShowEdges: true})
```

---

## Theming, Motion, RTL & Accessibility

### ThemeScope & Dynamic Color

```go
th := md3e.ThemeFromSeed(ui.Hex("#6750A4"), md3e.ThemeOptions{
	Dark:         true,
	Variant:      tokens.Expressive,
	Contrast:     tokens.ContrastHigh,
	Motion:       motion.Expressive,
	ReduceMotion: false,
})
md3e.ThemeScope(c, th, func() {
	// Nested widgets and ui.Overlay popups inherit th automatically.
})
```

`ThemeFromSeed` computes a full HCT tonal palette, so cache the `*md3e.Theme` instance in application state and rebuild it only when seed, variant, contrast, or dark mode changes.

### MotionScheme

```go
spec := motion.SpecOf(motion.Expressive, motion.FastSpatial)
anim := motion.Resolve(motion.Expressive, motion.DefaultSpatial, c.Preferences().ReduceMotion)
```

> Always build spring animations via `motion` rather than writing `ui.SpringOptions{Damping: ζ}` directly: MyGo's `Damping` field is the physical viscous damping coefficient $c$, not the dimensionless damping ratio $\zeta$.

### Right-to-Left (RTL) & ReduceMotion

- Layouts use logical start/end methods (`PaddingStart`, `PaddingEnd`, `MarginStart`, `MarginEnd`, `AnchorTopStart`, etc.). Calling `.Direction(ui.RightToLeft)` mirrors layouts and propagates into `ui.Overlay` popups (dialogs, sheets, menus, pickers).
- When `ReduceMotion` is enabled on the theme or `c.Preferences()`, spring animations settle immediately with zero pending frames (`tt.FramePending() == false`), `LoadingIndicator` holds its initial shape, and indeterminate progress indicators display static geometry.

---

## Testing & Verification

All UI tests use `ui.NewTester` with MyGo's deterministic virtual clock (no `time.Sleep`, no OS window required):

```go
tt := ui.NewTester(app.view, 1280, 800)
tt.Advance(300 * time.Millisecond)
if tt.FramePending() {
	// animation still settling
}
```

`golden_test.go` locks text-free geometric renderings (`testdata/<name>.png`, and `testdata/<name>_<GOOS>.png` on non-Windows hosts when present) via `github.com/egoist/mygo/ui/golden` to avoid cross-platform font rasterization drift:

```sh
# Normal verification (never leaves *.got.png behind)
CGO_ENABLED=0 go test -count=1 ./...

# Regenerate golden baselines only when intentionally changing visual output
MYGO_UPDATE_GOLDEN=1 CGO_ENABLED=0 go test -count=1 -run TestGolden .
```

### Full Check & Cross-Compilation Suite

```sh
# Build, static analysis, MyGo lifecycle vet (run without ./... args), and all tests
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
go tool mygo vet
CGO_ENABLED=0 go test -count=1 ./...

# Standalone M2 example module
cd examples/m2-demo && CGO_ENABLED=0 go test -count=1 ./... && cd ../..

# Cross-platform builds
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=js      GOARCH=wasm  go build ./...
```

---

## Acknowledgments & License

- Built on [MyGo](https://github.com/egoist/mygo) (MIT License).
- Built-in 24×24 icon path geometry in `internal/mdicon` follows standard Material Design Icons (Apache-2.0).
- **mygo-md3e** is released under the [MIT License](LICENSE).
