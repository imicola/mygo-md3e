package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
	"github.com/imicola/mygo-md3e/symbols"
)

const (
	docsPaneTree    = 0
	docsPaneDoc     = 1
	docsPaneOutline = 2

	docStreamInterval     = 30 * time.Millisecond
	docStreamBytesPerTick = 36
)

type docEntry struct {
	id             string
	fileName       string
	title          string
	summary        string
	category       string
	badge          string
	readTime       string
	icon           md3e.IconDraw
	folderPath     []string // ancestor folder IDs from root down
	defaultContent string
	content        string
	edited         bool
}

type docFolder struct {
	id         string
	title      string
	open       bool
	filterOpen bool
	subfolders []*docFolder
	docIDs     []string
}

type docsState struct {
	folders []*docFolder
	docs    []docEntry
	docByID map[string]int

	activeDocID     string
	selectedTreeKey any
	treeFilter      string
	lastTreeFilter  string

	treeWidth       float32
	readerPct       float32
	showOutline     bool
	mediumOutline   bool
	compactPane     int
	outlineDepthIdx int // 0=All, 1=H1-H2
	showRawSource   bool

	streamFeeding  bool
	streamOffset   int
	lastStreamTick time.Time

	docState  docview.State
	docScroll ui.ScrollState
}

func newDocsState() *docsState {
	docs := []docEntry{
		{
			id:       "guides/getting-started",
			fileName: "Getting Started.md",
			title:    "Getting Started with MyGo MD3E",
			summary:  "Pure-Go Material Design 3 Expressive layering, ThemeScope, interactive tasks, and docview blocks.",
			category: "Guide",
			badge:    "Start",
			readTime: "3 min",
			icon:     symbols.Description.Draw,
			defaultContent: `# Getting Started with MyGo MD3E

Welcome to the **MyGo MD3E Knowledge Workbench**. Every pane on this screen—the hierarchical ` + "`TreeView`" + ` directory on the left, the ` + "`docview.Markdown`" + ` reader in the center, and the scroll-linked ` + "`docview.Outline`" + ` on the right—is rendered natively in pure Go with zero WebView or HTML.

> [!NOTE]
> Use ` + "`md3e.ThemeScope`" + ` (or ` + "`AnimatedThemeScope`" + `) at the root of your window so nested components, ` + "`docview`" + ` blocks, and ` + "`ui.Overlay`" + ` popups automatically inherit the active HCT color scheme and density tokens.

## Package Layering

The library keeps strict acyclic dependencies so simple utilities never link heavy Markdown, Chart, or Video backends:

| Layer | Responsibility | Imports Root ` + "`md3e`" + ` |
| :--- | :--- | :--- |
| ` + "`tokens`" + ` | HCT / CAM16 color science, density, shape & type scales | No |
| ` + "`motion`" + ` | 12 Expressive & Standard spring presets (` + "`ζ -> c`" + `) | No |
| ` + "`shape`" + ` | 35 ` + "`RoundedPolygon`" + ` shapes & bijective Morph | No |
| ` + "`symbols`" + ` | 252 zero-bloat 24×24 icons with per-symbol DCE | No |
| ` + "`docview`" + ` | Markdown, streaming reveal, Callout, CodeBlock & Outline | Yes |

## Minimal Application Setup

Copy this snippet using the header button on the code card below to verify ` + "`docview.CodeBlock`" + ` clipboard integration and ` + "`SnackbarState`" + ` feedback:

` + "```go\n" + `package main

import (
	"github.com/egoist/mygo/ui"
	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

func appView(c *ui.Context, th *md3e.Theme) ui.Element {
	return md3e.ThemeScope(c, th, func() {
		ui.Column(c).Fill().Padding(24).Gap(12).Children(func() {
			md3e.Text(c, "Hello MD3E", tokens.HeadlineMedium, true)
			md3e.Button(c.Key("start"), "Launch", md3e.ButtonOptions{
				Variant: md3e.ButtonFilled,
			})
		})
	}).Fill()
}
` + "```\n\n" + `> [!TIP]
> Cross-reference other notes directly inside Markdown—for example jump to [Motion & Springs](doc://tokens/motion-physics) or inspect [Adaptive Shell](doc://guides/adaptive-shell).

## Interactive Checklist

Click any task checkbox below to update the underlying Markdown source live via ` + "`OnTask`" + `:

- [x] Mount ` + "`AdaptiveScaffold`" + ` with ` + "`WindowTitleBar`" + `
- [ ] Enable live ` + "`MarkdownStream`" + ` playback in the toolbar
- [ ] Toggle ` + "`Source`" + ` to inspect line-numbered Markdown source

## Keyboard & Selection

Select prose across paragraphs, tables, and callouts and press ` + "`Cmd+Shift+C`" + ` (` + "`Ctrl+Shift+C`" + ` on Linux/Windows) or right-click to use **Copy as Markdown**.
`,
		},
		{
			id:       "guides/adaptive-shell",
			fileName: "Adaptive Shell.md",
			title:    "Adaptive Multi-Pane Layouts",
			summary:  "How WindowSizeClass, AdaptiveScaffold, ListDetailPane, and SupportingPane adapt across breakpoints.",
			category: "Layout",
			readTime: "4 min",
			icon:     symbols.Article.Draw,
			defaultContent: `# Adaptive Multi-Pane Layouts

Material Design 3 defines three canonical breakpoints—**Compact** (` + "`< 600` DIP" + `), **Medium** (` + "`600..839` DIP" + `), and **Expanded** (` + "`>= 840` DIP" + `).

> [!IMPORTANT]
> Inside ` + "`AdaptiveScaffold`" + `, nested ` + "`BentoGrid`" + ` and ` + "`SupportingPane`" + ` components consult ` + "`md3e.ContentSizeClassOf(c)`" + ` (the width remaining beside the ` + "`NavigationRail`" + `) instead of the outer window width, preventing cramped columns.

## Composing Three Panes

Nesting ` + "`md3e.SupportingPane`" + ` inside the detail slot of ` + "`md3e.ListDetailPane`" + ` creates a full IDE or knowledge workbench layout with draggable split handles:

` + "```go\n" + `md3e.ListDetailPane(c.Key("workspace"), treePane, func() {
	md3e.SupportingPane(c.Key("reader"), docPane, outlinePane, md3e.SupportingPaneOptions{
		ShowSupporting:     &showOutline,
		DefaultMainPercent: 72,
	})
}, md3e.ListDetailPaneOptions{
	SplitWidth:       &treeWidth,
	DefaultListWidth: 232,
	ShowDetail:       &showDetail,
})
` + "```\n\n" + `## RTL & Compact Degradation

When ` + "`ui.RightToLeft`" + ` is active:

1. ` + "`NavigationRail`" + ` anchors to the right edge of the window.
2. ` + "`ListDetailPane`" + ` and ` + "`SupportingPane`" + ` mirror split direction cleanly.
3. ` + "`TreeView`" + ` chevrons, indent guides, and ` + "`Outline`" + ` active bars flip to the Start (right) edge.

### Breakpoint Matrix

| Width | Navigation | Tree & Reader | Outline |
| :--- | :--- | :--- | :--- |
| Expanded (` + "`>= 840`" + `) | Expanded ` + "`NavigationRail`" + ` | Side-by-side split | Visible ` + "`SupportingPane`" + ` |
| Medium (` + "`600..839`" + `) | Collapsed ` + "`NavigationRail`" + ` | Side-by-side split | Toggleable via chip |
| Compact (` + "`< 600`" + `) | Bottom ` + "`NavigationBar`" + ` | Single pane + back bar | Single pane + back bar |
`,
		},
		{
			id:       "guides/architecture/zero-bloat-dce",
			fileName: "Zero-Bloat & DCE.md",
			title:    "Zero-Bloat Architecture & Performance",
			summary:  "Lazy symbol parsing, reusable ui.Path buffers, and off-screen zero-frame sleeping.",
			category: "Performance",
			badge:    "P2",
			readTime: "4 min",
			icon:     symbols.Speed.Draw,
			defaultContent: `# Zero-Bloat Architecture & Performance

Every package in ` + "`mygo-md3e`" + ` is engineered so simple apps pay zero binary or memory overhead for unused subsystems.

> [!WARNING]
> Never add package-level ` + "`init()`" + ` component registries or ` + "`sync.OnceValue`" + ` closures referencing heavy shape/symbol catalogs—Go's linker cannot dead-code-eliminate symbols captured by package initializers.

## Reusable Path Buffers

Custom shape and chart drawing hot paths write directly into ` + "`ui.Local`" + `-cached ` + "`ui.Path`" + ` instances via ` + "`RoundedPolygon.WriteTo`" + ` and ` + "`shape.ClipShapeTo`" + `, achieving **0 allocs/op** in steady state.

` + "```go\n" + `scratch := ui.Local(el, pathKey{}, func() ui.Path { return ui.Path{} })
poly.WriteTo(scratch, r)
p.FillPath(scratch, cs.PrimaryContainer)
` + "```\n\n" + `## Viewport Visibility Sleep

Animated widgets (` + "`LinearWavyProgress`" + `, ` + "`LoadingIndicator`" + `, ` + "`StatusPill`" + `, ` + "`AreaSparkline`" + `) intersect their bounds against ` + "`p.Visible()`" + ` before requesting ` + "`p.AnimationFrame()`" + ` or ` + "`p.After(...)`" + `. Once scrolled off screen, CPU usage drops to **0%**.

> [!CAUTION]
> Inside ` + "`Draw`" + ` or ` + "`DrawOver`" + ` callbacks during ` + "`ElementTransition`" + ` exit fades, ` + "`el.Valid()`" + ` is ` + "`false`" + `. Always guard ` + "`Hovered()`" + `, ` + "`Pressed()`" + `, and ` + "`p.AnimationFrame()`" + ` calls behind ` + "`if el.Valid()`" + `.

## Steady-State Targets

| Benchmark | Target | Measured (Linux/amd64) |
| :--- | :--- | :--- |
| Dashboard Steady Frame | ` + "`< 0.5 ms`" + ` | ` + "`89 allocs/frame`" + ` |
| 100K DataTable Scroll | ` + "`< 0.3 ms`" + ` | ` + "`175 allocs/frame`" + ` |
| Retained Go Heap | ` + "`< 5 MB`" + ` | ` + "`~0.45 MB`" + ` |
`,
		},
		{
			id:       "tokens/hct-color",
			fileName: "HCT Color System.md",
			title:    "HCT Color Science & Dynamic Extraction",
			summary:  "CAM16 hue/chroma paired with CIE L*a*b* perceptual tone and image seed extraction.",
			category: "Tokens",
			readTime: "3 min",
			icon:     symbols.Palette.Draw,
			defaultContent: `# HCT Color Science & Dynamic Extraction

MD3E derives all 45+ semantic color roles from a single seed color in **HCT** (Hue, Chroma, Tone) space, ensuring predictable WCAG contrast across Light, Dark, and High Contrast themes.

## Surface Elevation Hierarchy

Instead of heavy drop shadows alone, MD3E separates stacked surfaces with five tonal container steps:

- ` + "`SurfaceContainerLowest`" + ` — deepest recessed canvas (` + "`Tone 4`" + ` dark / ` + "`Tone 100`" + ` light)
- ` + "`SurfaceContainerLow`" + ` — sidebars, navigation rails & reading panels
- ` + "`SurfaceContainer`" + ` — standard card groups
- ` + "`SurfaceContainerHigh`" + ` — code cards, tooltips & active menus
- ` + "`SurfaceContainerHighest`" + ` — badges, pills & focused tracks

> [!TIP]
> Pair ` + "`tokens.SeedFromImage`" + ` or ` + "`tokens.PaletteFromImage`" + ` with ` + "`md3e.AnimatedThemeScope`" + ` to smoothly transition an entire workspace when switching album covers or hero media.

## Theme Configuration Snapshot

` + "```json\n" + `{
  "seed": "#E06D1B",
  "variant": "Expressive",
  "density": "Desktop",
  "contrast": "Standard",
  "dark": true,
  "motion": "Expressive"
}
` + "```\n",
		},
		{
			id:       "tokens/motion-physics",
			fileName: "Motion & Springs.md",
			title:    "Expressive Spring Physics",
			summary:  "Converting MD3E damping ratios into MyGo viscous damping coefficients across 12 tokens.",
			category: "Tokens",
			readTime: "3 min",
			icon:     symbols.Speed.Draw,
			defaultContent: `# Expressive Spring Physics

Material Design 3 Expressive splits motion into **Spatial** (position, scale, corner morphing with playful overshoot) and **Effects** (color, opacity, state layers with critical damping ` + "`ζ = 1.0`" + `).

> [!IMPORTANT]
> MyGo's ` + "`ui.SpringOptions.Damping`" + ` field expects the **viscous damping coefficient** ` + "`c = 2 · ζ · √(k · m)`" + `, NOT the dimensionless damping ratio ` + "`ζ`" + `. Always construct animations via the ` + "`motion`" + ` package rather than hand-writing ` + "`ui.SpringOptions`" + `.

## Expressive vs Standard Presets

| Category | Speed | ` + "`ζ`" + ` (Expressive) | ` + "`k`" + ` (Expressive) | Settling Time |
| :--- | :--- | :--- | :--- | :--- |
| Spatial | Fast | ` + "`0.6`" + ` | ` + "`800`" + ` | ~420 ms |
| Spatial | Default | ` + "`0.8`" + ` | ` + "`380`" + ` | ~475 ms |
| Spatial | Slow | ` + "`0.8`" + ` | ` + "`200`" + ` | ~655 ms |
| Effects | Fast | ` + "`1.0`" + ` | ` + "`3800`" + ` | ~149 ms |
| Effects | Default | ` + "`1.0`" + ` | ` + "`1600`" + ` | ~230 ms |

## ReduceMotion Compliance

` + "```go\n" + `anim := th.Animation(motion.FastSpatial)
if th.ReduceMotion || c.Preferences().ReduceMotion {
	val = el.Jump(animKey{}, target)
} else {
	val = el.AnimateTo(animKey{}, target, anim)
}
` + "```\n",
		},
		{
			id:       "ai-notes/streaming-brief",
			fileName: "Streaming Brief.md",
			title:    "Streaming Assistant Brief",
			summary:  "Demonstrates docview.MarkdownStream incremental token healing, word-fade reveal, and live Outline updates.",
			category: "AI Notes",
			badge:    "Live",
			readTime: "3 min",
			icon:     symbols.AutoAwesome.Draw,
			defaultContent: `# AI Architecture Brief

Click **Stream** in the document toolbar to watch ` + "`docview.MarkdownStream`" + ` reveal this brief incrementally while the right-hand ` + "`docview.Outline`" + ` populates headings live as they arrive.

> [!NOTE]
> While a fenced code block is still streaming open (` + "`b.Open == true`" + `), ` + "`docview.MarkdownStream`" + ` automatically disables the Copy button until the closing fence arrives.

## Streaming Protocol

` + "`ui/markdown.Stream`" + ` heals incomplete syntax on the fly—unclosed emphasis spans, partial links, unfinished table rows, and open fenced blocks never flicker into raw Markdown tokens.

### Chunk Healing Guarantees

- Words fade smoothly over ` + "`300 ms`" + ` when motion is enabled.
- Under ` + "`ReduceMotion`" + `, text snaps via ` + "`markdown.AtOnce`" + ` with zero pending animation frames.
- Once complete, switching seamlessly to ` + "`docview.Markdown`" + ` preserves selection and scroll offsets.

## Telemetry Summary

` + "```go\n" + `opts := docview.MarkdownOptions{
	State:        &docState,
	Scroll:       &docScroll,
	HeadingRules: true,
	Snackbar:     snack,
}
if streaming {
	docview.MarkdownStream(c.Key("doc"), partialSrc, opts)
} else {
	docview.Markdown(c.Key("doc"), fullSrc, opts)
}
` + "```\n\n" + `> [!TIP]
> Try scrolling partway through the document and clicking sections inside ` + "`docview.Outline`" + `—the Primary start-edge bar glides smoothly to track your active heading.

## Verification Checklist

- [x] Incremental token append without full-document reparse
- [x] Live heading extraction synced to ` + "`docview.Outline`" + `
- [x] Virtual-clock ` + "`c.After`" + ` pacing with ` + "`FramePending() == false`" + ` at rest
`,
		},
		{
			id:       "reference/callout-matrix",
			fileName: "Callout & Code Matrix.md",
			title:    "Callout & Syntax Theme Matrix",
			summary:  "All five MD3E semantic GitHub Alert Callouts alongside multi-language syntax highlighting.",
			category: "Reference",
			readTime: "3 min",
			icon:     symbols.Code.Draw,
			defaultContent: `# Callout & Syntax Theme Matrix

` + "`docview`" + ` maps all five GitHub Alert quote variants to semantic MD3E ` + "`Callout`" + ` containers with harmonized contrast and RTL-mirrored accent stripes.

## Semantic Callout Variants

> [!NOTE]
> **Note (` + "`CalloutNote`" + `)** — Informational guidance mapped to ` + "`PrimaryContainer`" + `.

> [!TIP]
> **Tip (` + "`CalloutTip`" + `)** — Helpful workflow shortcuts mapped to ` + "`SuccessContainer`" + `.

> [!IMPORTANT]
> **Important (` + "`CalloutImportant`" + `)** — Key architectural constraints mapped to ` + "`TertiaryContainer`" + `.

> [!WARNING]
> **Warning (` + "`CalloutWarning`" + `)** — Breaking behavior or performance traps mapped to ` + "`WarningContainer`" + `.

> [!CAUTION]
> **Caution (` + "`CalloutCaution`" + `)** — High-risk or destructive operations mapped to ` + "`ErrorContainer`" + `.

## Multi-Language Highlighting

` + "`codeview.ThemeFromMD3E`" + ` guarantees >= 28° perceptual hue separation against the active seed color across Light, Dark, and High Contrast themes.

` + "```bash\n" + `CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
go tool mygo vet
CGO_ENABLED=0 go test -count=1 ./...
` + "```\n",
		},
		{
			id:       "reference/quick-tip",
			fileName: "Quick Tip.md",
			title:    "Quick Reference Card (No Headings)",
			summary:  "Compact reference note without headings, demonstrating Outline EmptyText fallback.",
			category: "Reference",
			readTime: "1 min",
			icon:     symbols.History.Draw,
			defaultContent: `This quick note intentionally contains **no Markdown headings**, demonstrating how ` + "`docview.Outline`" + ` gracefully renders its ` + "`EmptyText`" + ` state while the document remains interactive.

> [!TIP]
> Use ` + "`GOARCH=wasm GOOS=js go build ./...`" + ` to verify pure-Go browser compatibility alongside desktop builds.

` + "```sh\n" + `CGO_ENABLED=0 go run ./examples/workbench
` + "```\n",
		},
	}

	docByID := make(map[string]int, len(docs))
	for i := range docs {
		docs[i].content = docs[i].defaultContent
		docByID[docs[i].id] = i
	}

	folders := []*docFolder{
		{
			id:         "guides",
			title:      "Guides",
			open:       true,
			filterOpen: true,
			docIDs:     []string{"guides/getting-started", "guides/adaptive-shell"},
			subfolders: []*docFolder{
				{
					id:         "guides/architecture",
					title:      "Architecture",
					open:       true,
					filterOpen: true,
					docIDs:     []string{"guides/architecture/zero-bloat-dce"},
				},
			},
		},
		{
			id:         "tokens",
			title:      "Design Tokens",
			open:       true,
			filterOpen: true,
			docIDs:     []string{"tokens/hct-color", "tokens/motion-physics"},
		},
		{
			id:         "ai-notes",
			title:      "AI Notes",
			open:       true,
			filterOpen: true,
			docIDs:     []string{"ai-notes/streaming-brief"},
		},
		{
			id:         "reference",
			title:      "Reference",
			open:       false, // collapsed initially to showcase interactive TreeView expansion
			filterOpen: true,
			docIDs:     []string{"reference/callout-matrix", "reference/quick-tip"},
		},
	}

	// Populate ancestor folder paths on each document for Breadcrumb navigation.
	var assignPaths func(f *docFolder, path []string)
	assignPaths = func(f *docFolder, path []string) {
		nextPath := append(append([]string(nil), path...), f.id)
		for _, id := range f.docIDs {
			if idx, ok := docByID[id]; ok {
				docs[idx].folderPath = nextPath
			}
		}
		for _, sub := range f.subfolders {
			assignPaths(sub, nextPath)
		}
	}
	for _, f := range folders {
		assignPaths(f, nil)
	}

	initialID := docs[0].id
	return &docsState{
		folders:         folders,
		docs:            docs,
		docByID:         docByID,
		activeDocID:     initialID,
		selectedTreeKey: initialID,
		treeWidth:       232,
		readerPct:       72,
		showOutline:     true,
		compactPane:     docsPaneDoc,
		streamOffset:    len(docs[0].content),
	}
}

func (d *docsState) activeDoc() *docEntry {
	if idx, ok := d.docByID[d.activeDocID]; ok && idx >= 0 && idx < len(d.docs) {
		return &d.docs[idx]
	}
	if len(d.docs) > 0 {
		return &d.docs[0]
	}
	return nil
}

func (d *docsState) activeDocIndex() int {
	if idx, ok := d.docByID[d.activeDocID]; ok && idx >= 0 && idx < len(d.docs) {
		return idx
	}
	return 0
}

func (d *docsState) findFolder(id string) *docFolder {
	var walk func([]*docFolder) *docFolder
	walk = func(list []*docFolder) *docFolder {
		for _, f := range list {
			if f.id == id {
				return f
			}
			if sub := walk(f.subfolders); sub != nil {
				return sub
			}
		}
		return nil
	}
	return walk(d.folders)
}

func (d *docsState) setAllFoldersOpen(open bool) {
	var walk func([]*docFolder)
	walk = func(list []*docFolder) {
		for _, f := range list {
			f.open = open
			f.filterOpen = open
			walk(f.subfolders)
		}
	}
	walk(d.folders)
}

func (d *docsState) expandPath(folderPath []string) {
	for _, fid := range folderPath {
		if f := d.findFolder(fid); f != nil {
			f.open = true
			f.filterOpen = true
		}
	}
}

func (d *docsState) syncFilterOpen() {
	if d.treeFilter != d.lastTreeFilter {
		d.lastTreeFilter = d.treeFilter
		var reset func([]*docFolder)
		reset = func(list []*docFolder) {
			for _, f := range list {
				f.filterOpen = true
				reset(f.subfolders)
			}
		}
		reset(d.folders)
	}
}

func (d *docsState) selectDoc(id string) bool {
	idx, ok := d.docByID[id]
	if !ok || idx < 0 || idx >= len(d.docs) {
		return false
	}
	doc := &d.docs[idx]
	d.activeDocID = doc.id
	d.selectedTreeKey = doc.id
	d.expandPath(doc.folderPath)
	d.streamFeeding = false
	d.lastStreamTick = time.Time{}
	d.streamOffset = len(doc.content)
	d.docScroll = ui.ScrollState{}
	return true
}

func (d *docsState) stepDoc(delta int) bool {
	if len(d.docs) == 0 {
		return false
	}
	next := d.activeDocIndex() + delta
	if next < 0 || next >= len(d.docs) {
		return false
	}
	return d.selectDoc(d.docs[next].id)
}

func (d *docsState) startStream(reduce bool) {
	doc := d.activeDoc()
	if doc == nil {
		return
	}
	d.showRawSource = false
	d.docScroll = ui.ScrollState{}
	if reduce || len(doc.content) == 0 {
		d.streamFeeding = false
		d.lastStreamTick = time.Time{}
		d.streamOffset = len(doc.content)
		return
	}
	d.streamFeeding = true
	d.streamOffset = 0
	d.lastStreamTick = time.Time{}
	d.advanceStreamBytes(docStreamBytesPerTick)
}

func (d *docsState) stopStream() {
	d.streamFeeding = false
	d.lastStreamTick = time.Time{}
}

func (d *docsState) completeStream() {
	doc := d.activeDoc()
	d.streamFeeding = false
	d.lastStreamTick = time.Time{}
	if doc != nil {
		d.streamOffset = len(doc.content)
	}
}

func (d *docsState) pauseStream() {
	d.lastStreamTick = time.Time{}
}

func (d *docsState) advanceStreamBytes(n int) {
	doc := d.activeDoc()
	if doc == nil || n <= 0 {
		d.streamFeeding = false
		return
	}
	src := doc.content
	if d.streamOffset >= len(src) {
		d.streamOffset = len(src)
		d.streamFeeding = false
		d.lastStreamTick = time.Time{}
		return
	}
	target := min(len(src), d.streamOffset+n)
	for target < len(src) && !utf8.RuneStart(src[target]) {
		target++
	}
	// Snap to end of the word/whitespace chunk so tokens arrive smoothly.
	for target < len(src) && target < d.streamOffset+n+12 && src[target] != ' ' && src[target] != '\n' {
		_, w := utf8.DecodeRuneInString(src[target:])
		target += w
	}
	d.streamOffset = min(len(src), target)
	if d.streamOffset >= len(src) {
		d.streamOffset = len(src)
		d.streamFeeding = false
		d.lastStreamTick = time.Time{}
	}
}

func (d *docsState) advanceIfDue(now time.Time) {
	if !d.streamFeeding {
		d.lastStreamTick = time.Time{}
		return
	}
	if d.lastStreamTick.IsZero() {
		d.lastStreamTick = now
		d.advanceStreamBytes(docStreamBytesPerTick)
		return
	}
	elapsed := now.Sub(d.lastStreamTick)
	if elapsed < docStreamInterval {
		return
	}
	steps := min(int(elapsed/docStreamInterval), 64)
	for range steps {
		if !d.streamFeeding {
			break
		}
		d.advanceStreamBytes(docStreamBytesPerTick)
	}
	if d.streamFeeding {
		d.lastStreamTick = now
	} else {
		d.lastStreamTick = time.Time{}
	}
}

func (d *docsState) scheduleNext(c *ui.Context, now time.Time) {
	if !d.streamFeeding {
		d.lastStreamTick = time.Time{}
		return
	}
	if d.lastStreamTick.IsZero() {
		d.lastStreamTick = now
	}
	rem := docStreamInterval - now.Sub(d.lastStreamTick)
	if rem <= 0 {
		rem = docStreamInterval
	}
	c.After(rem)
}

func (d *docsState) renderedMarkdown() string {
	doc := d.activeDoc()
	if doc == nil {
		return ""
	}
	if d.streamFeeding || d.streamOffset < len(doc.content) {
		off := clampInt(d.streamOffset, 0, len(doc.content))
		return doc.content[:off]
	}
	return doc.content
}

func (d *docsState) streamProgress() float64 {
	doc := d.activeDoc()
	if doc == nil || len(doc.content) == 0 {
		return 1
	}
	if !d.streamFeeding && d.streamOffset >= len(doc.content) {
		return 1
	}
	return clamp01(float64(d.streamOffset) / float64(len(doc.content)))
}

func (d *docsState) toggleActiveTask(line1Based int, checked bool) bool {
	doc := d.activeDoc()
	if doc == nil {
		return false
	}
	next, ok := toggleMarkdownTaskLine(doc.content, line1Based, checked)
	if !ok {
		return false
	}
	doc.content = next
	doc.edited = doc.content != doc.defaultContent
	if !d.streamFeeding {
		d.streamOffset = len(doc.content)
	} else if d.streamOffset > len(doc.content) {
		d.streamOffset = len(doc.content)
	}
	return true
}

func (d *docsState) resetActiveDoc() bool {
	doc := d.activeDoc()
	if doc == nil || !doc.edited {
		return false
	}
	doc.content = doc.defaultContent
	doc.edited = false
	if !d.streamFeeding {
		d.streamOffset = len(doc.content)
	}
	return true
}

func toggleMarkdownTaskLine(src string, line1Based int, checked bool) (string, bool) {
	if line1Based < 1 {
		return src, false
	}
	lines := strings.Split(src, "\n")
	idx := line1Based - 1
	if idx >= len(lines) {
		return src, false
	}
	line := lines[idx]
	target := "[ ]"
	repl := "[x]"
	if !checked {
		target = "[x]"
		repl = "[ ]"
	}
	pos := strings.Index(line, target)
	if pos < 0 && !checked {
		pos = strings.Index(line, "[X]")
	}
	if pos < 0 {
		return src, false
	}
	lines[idx] = line[:pos] + repl + line[pos+3:]
	return strings.Join(lines, "\n"), true
}

func (d *docsState) docMatchesFilter(doc *docEntry, q string) bool {
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(doc.title), q) ||
		strings.Contains(strings.ToLower(doc.fileName), q) ||
		strings.Contains(strings.ToLower(doc.category), q) ||
		strings.Contains(strings.ToLower(doc.summary), q) ||
		strings.Contains(strings.ToLower(doc.content), q) {
		return true
	}
	for _, fid := range doc.folderPath {
		if f := d.findFolder(fid); f != nil && strings.Contains(strings.ToLower(f.title), q) {
			return true
		}
	}
	return false
}

func (d *docsState) buildTreeNodes(onDocClick func()) ([]md3e.TreeNode, int) {
	d.syncFilterOpen()
	q := strings.ToLower(strings.TrimSpace(d.treeFilter))
	matchedCount := 0

	var buildFolder func(f *docFolder) (md3e.TreeNode, int, bool)
	buildFolder = func(f *docFolder) (md3e.TreeNode, int, bool) {
		var children []md3e.TreeNode
		folderMatches := q != "" && strings.Contains(strings.ToLower(f.title), q)
		subQ := q
		if folderMatches {
			subQ = ""
		}

		count := 0
		for _, sub := range f.subfolders {
			if subNode, subCount, ok := buildFolder(sub); ok {
				children = append(children, subNode)
				count += subCount
			}
		}
		for _, docID := range f.docIDs {
			idx, ok := d.docByID[docID]
			if !ok {
				continue
			}
			doc := &d.docs[idx]
			if !d.docMatchesFilter(doc, subQ) {
				continue
			}
			idCopy := doc.id
			badge := doc.badge
			if doc.edited {
				badge = "Edited"
			}
			children = append(children, md3e.TreeNode{
				Key:   doc.id,
				Label: doc.fileName,
				Icon:  doc.icon,
				Badge: badge,
				OnClick: func() {
					d.selectDoc(idCopy)
					if onDocClick != nil {
						onDocClick()
					}
				},
			})
			count++
			matchedCount++
		}

		if len(children) == 0 && q != "" && !folderMatches {
			return md3e.TreeNode{}, 0, false
		}

		openPtr := &f.open
		if q != "" {
			openPtr = &f.filterOpen
		}
		icon := symbols.Folder.Draw
		if *openPtr {
			icon = symbols.FolderOpen.Draw
		}
		return md3e.TreeNode{
			Key:      "folder:" + f.id,
			Label:    f.title,
			Icon:     icon,
			Badge:    fmt.Sprintf("%d", count),
			Open:     openPtr,
			Children: children,
			OnClick: func() {
				*openPtr = !*openPtr
			},
		}, count, true
	}

	nodes := make([]md3e.TreeNode, 0, len(d.folders))
	for _, f := range d.folders {
		if node, _, ok := buildFolder(f); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes, matchedCount
}
