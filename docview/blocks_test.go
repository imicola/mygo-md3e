package docview_test

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestCodeBlockCopyFeedbackAndSnackbar(t *testing.T) {
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	snack := &md3e.SnackbarState{}
	var copied string
	src := "package main\n\nfunc main() {\n\tprintln(42)\n}\n"

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Children(func() {
				docview.CodeBlock(c.Key("go-code"), src, docview.CodeBlockOptions{
					Language:        "go",
					Title:           "main.go",
					ShowLineNumbers: true,
					Snackbar:        snack,
					OnCopy:          func(s string) { copied = s },
				})
				md3e.SnackbarHost(c, snack)
			})
		}).Fill()
	}, 640, 360)

	if !tt.HasText("go") || !tt.HasText("main.go") || !tt.HasText("Copy") {
		t.Fatalf("missing code card chrome in %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Code, main.go (go)"); !ok {
		t.Fatal("missing accessible CodeBlock group")
	}

	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("click Copy: %v", err)
	}
	tt.Frame()
	if got := tt.Clipboard(); got != src {
		t.Fatalf("clipboard = %q, want %q", got, src)
	}
	if copied != src {
		t.Fatalf("OnCopy = %q, want %q", copied, src)
	}
	if !tt.HasText("Copied") {
		t.Fatalf("expected Copied state in %q", tt.Texts())
	}
	if !tt.HasText("Copied go code") {
		t.Fatalf("expected Snackbar confirmation in %q", tt.Texts())
	}

	// After 2 seconds the button returns to Copy while the 4s Snackbar settles cleanly.
	tt.Advance(2100 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copy go code"); !ok {
		t.Fatalf("expected Copy button to revert after 2s, texts=%q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copied"); ok {
		t.Fatalf("expected Copied button state to clear after 2s, texts=%q", tt.Texts())
	}
	tt.Advance(5 * time.Second)
	if tt.FramePending() {
		t.Fatal("expected zero pending frames after copy feedback and snackbar settled")
	}

	// Selecting all inside the code body copies code without line numbers.
	if r, ok := tt.Find("println(42)"); ok {
		tt.ClickAt(r.X+2, r.Y+r.H/2)
		tt.Command("selectAll")
		tt.Command("copy")
		gotSel := tt.Clipboard()
		if !strings.Contains(gotSel, "println(42)") || strings.Contains(gotSel, "1\n") {
			t.Fatalf("unexpected selected copy output: %q", gotSel)
		}
	}
}

func TestCodeBlockExitGhostSettling(t *testing.T) {
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	show := true
	exitTr := ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Children(func() {
				if show {
					ui.Box(c.Key("code-exit")).Transition(exitTr).Children(func() {
						docview.CodeBlock(c, "fmt.Println(1)", docview.CodeBlockOptions{Language: "go"})
					})
				}
			})
		}).Fill()
	}, 480, 220)

	_ = tt.Click("Copy")
	tt.Frame()
	show = false
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exited CodeBlock ghost left pending frames")
	}
}

func TestCodeBlockDisabledAndRTLLTRBody(t *testing.T) {
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true, Density: tokens.DensityCompact})
	copied := false

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Direction(ui.RightToLeft).Padding(16).Children(func() {
				docview.CodeBlock(c.Key("rtl-code"), "fmt.Println(\"ok\")", docview.CodeBlockOptions{
					Language: "go",
					Disabled: true,
					Wrap:     true,
					OnCopy:   func(string) { copied = true },
				})
			})
		}).Fill()
	}, 520, 260)

	btn, ok := tt.AccessByRole(ui.RoleButton, "Copy go code")
	if !ok || !btn.Disabled {
		t.Fatalf("expected disabled Copy button in accessibility tree, got ok=%v btn=%+v", ok, btn)
	}
	_ = tt.Click("Copy")
	tt.Frame()
	if copied {
		t.Fatal("disabled copy button invoked OnCopy")
	}
	// Language badge sits on Start (right side in RTL) while Copy sits on End (left side in RTL).
	langRect, ok1 := tt.Find("go")
	copyRect, ok2 := tt.Find("Copy")
	if !ok1 || !ok2 || !(copyRect.X < langRect.X) {
		t.Fatalf("expected RTL mirrored header (Copy left of badge), got copy=%+v lang=%+v", copyRect, langRect)
	}
}

func TestCalloutVariantsThemeAndRTL(t *testing.T) {
	kinds := []struct {
		kind  docview.CalloutKind
		title string
	}{
		{docview.CalloutNote, "Note"},
		{docview.CalloutTip, "Tip"},
		{docview.CalloutImportant, "Important"},
		{docview.CalloutWarning, "Warning"},
		{docview.CalloutCaution, "Caution"},
	}

	for _, dark := range []bool{false, true} {
		th := md3e.ThemeFromSeed(ui.Hex("#0061A4"), md3e.ThemeOptions{Dark: dark})
		tt := ui.NewTester(func(c *ui.Context) {
			md3e.ThemeScope(c, th, func() {
				ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
					for _, k := range kinds {
						docview.Callout(c.Key(k.title), k.kind, nil, docview.CalloutOptions{
							Text: "Details for " + k.title,
						})
					}
				})
			}).Fill()
		}, 560, 560)

		for _, k := range kinds {
			if !tt.HasText(k.title) || !tt.HasText("Details for "+k.title) {
				t.Fatalf("dark=%v: missing callout %q in %q", dark, k.title, tt.Texts())
			}
			if _, ok := tt.AccessByRole(ui.RoleRegion, k.title); !ok {
				t.Fatalf("dark=%v: missing RoleRegion for %q", dark, k.title)
			}
		}
	}

	// Verify RTL renders leading accent stripe on the right edge (Start edge in RTL).
	th := md3e.ThemeFromSeed(ui.Hex("#6750A4"))
	ttRTL := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Direction(ui.RightToLeft).Padding(16).Children(func() {
				docview.Callout(c.Key("caution-rtl"), docview.CalloutCaution, nil, docview.CalloutOptions{
					Title:     "Danger Zone",
					Text:      "Irreversible action",
					LiveAlert: true,
				})
			})
		}).Fill()
	}, 400, 160)

	reg, ok := ttRTL.AccessByRole(ui.RoleAlert, "Danger Zone")
	if !ok {
		t.Fatal("expected RoleAlert when LiveAlert=true")
	}
	img := ttRTL.Image()
	errCol := th.Colors.Error
	rightX := int(reg.Bounds.X + reg.Bounds.W - 2)
	leftX := int(reg.Bounds.X + 2)
	midY := int(reg.Bounds.Y + reg.Bounds.H/2)
	rightPx := img.RGBAAt(rightX, midY)
	leftPx := img.RGBAAt(leftX, midY)
	if rightPx.R != errCol.R || rightPx.G != errCol.G || rightPx.B != errCol.B {
		t.Fatalf("expected RTL accent stripe on right edge (%d,%d)=%v, want %v", rightX, midY, rightPx, errCol)
	}
	if leftPx.R == errCol.R && leftPx.G == errCol.G && leftPx.B == errCol.B {
		t.Fatalf("unexpected left-edge accent stripe in RTL (%d,%d)=%v", leftX, midY, leftPx)
	}

	// Out-of-range CalloutKind degrades safely to Note.
	if got := docview.CalloutKind(99).String(); got != "Note" {
		t.Fatalf("out-of-range CalloutKind.String() = %q, want Note", got)
	}
}

func TestCodeBlockBorderlessDefaultHairlineOptInAndHighContrast(t *testing.T) {
	darkStd := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true, ReduceMotion: true})
	darkHC := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
		Dark:         true,
		Contrast:     tokens.ContrastHigh,
		ReduceMotion: true,
	})

	showCopy := false
	renderBottomEdge := func(th *md3e.Theme, hairline *bool) (r, g, b uint8) {
		tt := ui.NewTester(func(c *ui.Context) {
			md3e.ThemeScope(c, th, func() {
				ui.Box(c).Size(320, 140).Background(th.Colors.Surface).Padding(20).Children(func() {
					docview.CodeBlock(c.Key("cb"), "println(1)", docview.CodeBlockOptions{
						ShowCopy: &showCopy,
						Hairline: hairline,
					})
				})
			})
		}, 320, 140)
		tt.Frame()
		grp, ok := tt.AccessByRole(ui.RoleGroup, "Code")
		if !ok {
			t.Fatal("CodeBlock missing RoleGroup")
		}
		// Top center edge of borderless header-free CodeBlock.
		px := tt.Image().RGBAAt(int(grp.Bounds.X+grp.Bounds.W/2), int(grp.Bounds.Y))
		return px.R, px.G, px.B
	}

	absDiff := func(a, b uint8) int {
		d := int(a) - int(b)
		if d < 0 {
			return -d
		}
		return d
	}

	wantBg := darkStd.Colors.SurfaceContainer
	r0, g0, b0 := renderBottomEdge(darkStd, nil)
	if absDiff(r0, wantBg.R) > 4 || absDiff(g0, wantBg.G) > 4 || absDiff(b0, wantBg.B) > 4 {
		t.Fatalf("default CodeBlock (Hairline==nil) drew unexpected border or wrong bg: top=(%d,%d,%d) want SurfaceContainer=(%d,%d,%d)",
			r0, g0, b0, wantBg.R, wantBg.G, wantBg.B)
	}

	wantHair := true
	r1, g1, b1 := renderBottomEdge(darkStd, &wantHair)
	if absDiff(r1, wantBg.R) <= 4 && absDiff(g1, wantBg.G) <= 4 && absDiff(b1, wantBg.B) <= 4 {
		t.Fatalf("CodeBlock with explicit Hairline=true did not paint border: top=(%d,%d,%d)", r1, g1, b1)
	}

	hcOut := darkHC.Colors.Outline
	r2, g2, b2 := renderBottomEdge(darkHC, nil)
	if absDiff(r2, hcOut.R) > 35 || absDiff(g2, hcOut.G) > 35 || absDiff(b2, hcOut.B) > 35 {
		t.Fatalf("HighContrast CodeBlock did not paint 1.5 DIP Outline border: top=(%d,%d,%d) want~=(%d,%d,%d)",
			r2, g2, b2, hcOut.R, hcOut.G, hcOut.B)
	}
}
