package md3e

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// WindowController is the minimal native window interface satisfied directly by
// *mygo.Window without coupling package md3e to the platform window backend.
// When the bound window also implements the optional
// SetWindowButtonsTheme(mygo.ThemeSource) method (as *mygo.Window does),
// WindowTitleBar automatically syncs native system caption button appearance
// with the active light/dark theme whenever system buttons are visible.
type WindowController interface {
	Minimize()
	ToggleMaximize()
	Close()
	IsMaximized() bool
	SetAlwaysOnTop(v bool)
}

type windowButtonsThemeController interface {
	SetWindowButtonsTheme(mygo.ThemeSource)
}

// WindowTitleBarOptions configures WindowTitleBar. The zero value renders a
// seamlessly integrated SurfaceContainerLowest title bar whose height adapts
// to c.TitleBar() and Theme.Density, reserves platform safe insets (macOS
// traffic lights / Windows & Linux system controls), and enables window
// dragging and double-click maximize via DragWindow().
//
// For desktop apps on Windows and Linux, pairing mygo.WindowOptions{
// TitleBarStyle: mygo.TitleBarHidden, HideWindowButtons: true} with
// ShowWindowControls: true is recommended so WindowTitleBar renders 100%
// MD3E-styled, height-aligned Minimize, Maximize/Restore (integrated with
// Windows 11 Snap Layouts), and Close controls with rounded hover highlights.
// Keep HideWindowButtons: false on macOS to preserve native traffic lights.
type WindowTitleBarOptions struct {
	// Title is the window title shown in the center region when Center is nil.
	Title string
	// Height overrides the minimum bar height in DIP. Actual height is always
	// at least c.TitleBar().Height so system controls never clip.
	Height float32
	// Background overrides the title bar fill. When zero, uses
	// SurfaceColorForElevation(th, Elevation) if Elevation > Level0, or
	// SurfaceContainerLowest otherwise.
	Background ui.Color
	// Elevation optionally raises the title bar tonal surface tier.
	Elevation tokens.ElevationLevel
	// Divider draws a subtle 1 DIP hairline along the bottom edge of the bar.
	Divider bool
	// Label names the title bar landmark for assistive technology; defaults to
	// "Title bar".
	Label string

	// Leading builds optional content on the start side (after safe insets).
	Leading func() ui.Element
	// Center builds optional center content in place of Title.
	Center func() ui.Element
	// Trailing builds optional actions on the end side (before window controls).
	Trailing func() ui.Element

	// Window optionally binds a *mygo.Window (or custom WindowController) so
	// built-in Pin, Minimize, Maximize/Restore, and Close controls act directly
	// on the window when explicit callbacks are nil. When native system buttons
	// remain visible (HideWindowButtons: false, i.e. non-zero c.TitleBar()),
	// WindowTitleBar also inspects the optional SetWindowButtonsTheme method
	// to sync native caption glyph polarity with th.Dark. Safe to leave nil in tests.
	Window WindowController
	// ShowWindowControls renders MD3E Minimize, Maximize/Restore (marked
	// MaximizeButton for Windows 11 Snap Layouts), and Close buttons with
	// height-aligned rounded hover highlights. Recommended on non-macOS desktop
	// windows configured with TitleBarStyle: mygo.TitleBarHidden and
	// HideWindowButtons: true. Even when false, providing any individual
	// OnMinimize / OnMaximize / OnClose callback shows that control.
	ShowWindowControls bool

	// Pinned binds an always-on-top toggle button when non-nil (or when OnPin
	// is set). Clicking toggles *Pinned, updates Window.SetAlwaysOnTop when
	// Window is set, and calls OnPin.
	Pinned *bool
	// OnPin runs after the pin button toggles always-on-top state.
	OnPin func(pinned bool)

	// Maximized reflects the maximized visual state and accessible label
	// ("Restore" vs "Maximize") when Window is nil.
	Maximized bool
	// OnMinimize overrides the Minimize button action.
	OnMinimize func()
	// OnMaximize overrides the Maximize/Restore button action.
	OnMaximize func()
	// OnClose overrides the Close button action.
	OnClose func()
}

const titleBarNarrowThreshold float32 = 520

type (
	titleBarLeadingKey      struct{}
	titleBarTrailingKey     struct{}
	titleBarPinLocalKey     struct{}
	titleBarButtonsThemeKey struct{}
	titleBarPinKey          struct{}
	titleBarMinKey          struct{}
	titleBarMaxKey          struct{}
	titleBarCloseKey        struct{}
)

type titleBarButtonsThemeState struct {
	synced bool
	source mygo.ThemeSource
}

// WindowTitleBar builds a cross-platform MD3E desktop window title bar that
// respects c.TitleBar() safe insets, integrates seamlessly with
// SurfaceContainerLowest, supports window dragging and double-click maximize
// via DragWindow(), exposes Leading/Center/Trailing slots, automatically syncs
// native window button polarity via SetWindowButtonsTheme when system buttons
// are visible, gracefully collapses crowded centered Title text below 520 DIP,
// and optionally renders accessible MD3E Pin, Minimize, Maximize/Restore
// (marked MaximizeButton for Windows 11 Snap Layouts), and Close controls.
//
// Desktop recommendation: on Windows and Linux, set TitleBarStyle:
// mygo.TitleBarHidden and HideWindowButtons: true on mygo.WindowOptions with
// ShowWindowControls: true here; on macOS, keep HideWindowButtons: false so
// native traffic lights stay inside c.TitleBar().
func WindowTitleBar(c *ui.Context, opts ...WindowTitleBarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: WindowTitleBar takes at most one WindowTitleBarOptions")
	}
	var o WindowTitleBarOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	tb := c.TitleBar()

	minH := titleBarMinHeight(th.Density)
	if isPositiveFiniteF32(o.Height) {
		minH = o.Height
	}
	barH := max(minH, sanitizeNonNegF32(tb.Height))

	bg := o.Background
	if bg.A == 0 {
		if o.Elevation > tokens.Level0 {
			bg = SurfaceColorForElevation(th, o.Elevation)
		} else {
			bg = cs.SurfaceContainerLowest
			if bg.A == 0 {
				bg = cs.Surface
			}
		}
	}
	fg := th.ContentColorFor(bg)
	label := labelOr(o.Label, "Title bar")

	leftInset := sanitizeNonNegF32(tb.Left)
	rightInset := sanitizeNonNegF32(tb.Right)

	showPin := o.Pinned != nil || o.OnPin != nil
	showMin := o.ShowWindowControls || o.OnMinimize != nil
	showMax := o.ShowWindowControls || o.OnMaximize != nil
	showClose := o.ShowWindowControls || o.OnClose != nil
	hasWinControls := showMin || showMax || showClose

	root := ui.Row(c).
		FillWidth().
		Height(barH).
		Padding(0, rightInset, 0, leftInset).
		Background(bg).
		TextColor(fg).
		AlignItems(ui.Center).
		FocusRing(false).
		DragWindow().
		Role(ui.RoleBanner).
		Label(label)

	hasNativeButtons := tb.Height > 0 || tb.Left > 0 || tb.Right > 0 || tb.Bottom > 0
	themeState := ui.Local(root, titleBarButtonsThemeKey{}, func() titleBarButtonsThemeState {
		return titleBarButtonsThemeState{}
	})
	if o.Window != nil && hasNativeButtons {
		if setter, ok := o.Window.(windowButtonsThemeController); ok {
			darkButtons := th.Dark
			if o.Background.A > 0 {
				darkButtons = relativeLuminance(bg) <= 0.35
			}
			wantSource := mygo.ThemeLight
			if darkButtons {
				wantSource = mygo.ThemeDark
			}
			if !themeState.synced || themeState.source != wantSource {
				setter.SetWindowButtonsTheme(wantSource)
				themeState.synced = true
				themeState.source = wantSource
			}
		}
	} else {
		themeState.synced = false
	}

	if o.Divider {
		divCol := cs.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
			divCol = cs.Outline
		}
		if divCol.A > 0 {
			root.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
					return
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, divCol, 0)
			})
		}
	}

	pinState := o.Pinned
	if pinState == nil && showPin {
		pinState = ui.Local(root, titleBarPinLocalKey{}, func() bool { return false })
	}

	endPad := float32(8)
	if hasWinControls {
		endPad = 4
	}
	btnW := float32(42)
	if th.Density.Normalize() == tokens.DensityCompact {
		btnW = 36
	}
	pinSide := min(float32(30), max(float32(22), barH-8))
	if barH < 30 {
		pinSide = max(barH-4, 18)
	}
	titleRole := tokens.TitleSmall
	if th.Density.Normalize() != tokens.DensityComfortable {
		titleRole = tokens.LabelLarge
	}

	winW, _ := c.Size()
	hasStartChrome := o.Leading != nil || leftInset > 0
	hasEndChrome := o.Trailing != nil || showPin || hasWinControls || rightInset > 0
	hasMultipleEndSlots := (o.Trailing != nil && (showPin || hasWinControls || rightInset > 0)) ||
		(showPin && (hasWinControls || rightInset > 0))
	collapseCenterTitle := winW > 0 && winW < titleBarNarrowThreshold &&
		((hasStartChrome && hasEndChrome) || hasMultipleEndSlots)

	root.Children(func() {
		ui.Row(c).
			Grow(1).
			FillHeight().
			MinWidth(0).
			PaddingStart(8).
			PaddingEnd(endPad).
			Gap(8).
			AlignItems(ui.Center).
			DragWindow().
			Children(func() {
				if o.Leading != nil {
					ui.Row(c.Key(titleBarLeadingKey{})).
						Shrink(0).
						AlignItems(ui.Center).
						Gap(4).
						Children(func() {
							o.Leading()
						})
				}

				ui.Row(c).
					Grow(1).
					Shrink(1).
					FillHeight().
					MinWidth(0).
					Clip().
					AlignItems(ui.Center).
					Justify(ui.Center).
					DragWindow().
					Children(func() {
						if o.Center != nil {
							o.Center()
						} else if o.Title != "" && !collapseCenterTitle {
							txt := ui.Text(c, o.Title).SingleLine().MinWidth(0).MaxWidthPercent(100)
							applyTypeStyleWithTheme(th, txt, titleRole)
							txt.TextColor(fg)
						}
					})

				if o.Trailing != nil {
					ui.Row(c.Key(titleBarTrailingKey{})).
						Shrink(0).
						AlignItems(ui.Center).
						Gap(4).
						Children(func() {
							o.Trailing()
						})
				}

				if showPin {
					pinned := *pinState
					pinInk := cs.OnSurfaceVariant
					if pinned {
						pinInk = cs.Primary
					}
					btn := ui.ButtonBase(c.Key(titleBarPinKey{})).
						DimDisabled(false).
						FocusRing(false).
						Size(pinSide, pinSide).
						Shrink(0).
						Radius(8).
						AlignItems(ui.Center).
						Justify(ui.Center).
						Label("Pin window").
						AccessSelected(pinned)
					if pinned {
						btn.Background(cs.PrimaryContainer)
						pinInk = cs.OnPrimaryContainer
					}
					attachStateLayerWithTheme(th, btn, StateLayerOptions{
						Color:  pinInk,
						Radius: 8,
					})
					btn.Children(func() {
						ui.Box(c).Size(16, 16).Draw(func(p *ui.Painter, r ui.Rect) {
							drawTitleBarPinIcon(p, r, pinInk, pinned)
						})
					})
					if btn.Clicked() {
						next := !pinned
						*pinState = next
						if o.Window != nil {
							o.Window.SetAlwaysOnTop(next)
						}
						if o.OnPin != nil {
							o.OnPin(next)
						}
					}
				}

				if hasWinControls {
					ui.Row(c).Shrink(0).FillHeight().AlignItems(ui.Center).Children(func() {
						if showMin {
							btn := titleBarControlButton(c, th, titleBarMinKey{}, btnW, barH, "Minimize", fg, fg, func(p *ui.Painter, r ui.Rect, ink ui.Color) {
								drawTitleBarMinimizeIcon(p, r, ink)
							})
							if btn.Clicked() {
								if o.OnMinimize != nil {
									o.OnMinimize()
								} else if o.Window != nil {
									o.Window.Minimize()
								}
							}
						}
						if showMax {
							maximized := o.Maximized
							if o.Window != nil {
								maximized = o.Window.IsMaximized()
							}
							maxLabel := "Maximize"
							if maximized {
								maxLabel = "Restore"
							}
							btn := titleBarControlButton(c, th, titleBarMaxKey{}, btnW, barH, maxLabel, fg, fg, func(p *ui.Painter, r ui.Rect, ink ui.Color) {
								drawTitleBarMaximizeIcon(p, r, ink, maximized)
							})
							btn.MaximizeButton()
							if btn.Clicked() {
								if o.OnMaximize != nil {
									o.OnMaximize()
								} else if o.Window != nil {
									o.Window.ToggleMaximize()
								}
							}
						}
						if showClose {
							btn := titleBarControlButton(c, th, titleBarCloseKey{}, btnW, barH, "Close", fg, cs.Error, func(p *ui.Painter, r ui.Rect, ink ui.Color) {
								drawTitleBarCloseIcon(p, r, ink)
							})
							if btn.Clicked() {
								if o.OnClose != nil {
									o.OnClose()
								} else if o.Window != nil {
									o.Window.Close()
								}
							}
						}
					})
				}
			})
	})
	return root
}

func titleBarMinHeight(d tokens.Density) float32 {
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return 38
	case tokens.DensityCompact:
		return 34
	default:
		return 44
	}
}

func titleBarControlButton(c *ui.Context, th *Theme, key any, w, h float32, label string, ink, stateColor ui.Color, draw IconDraw) ui.Element {
	pillH := min(float32(30), max(float32(20), h-8))
	if h < 28 {
		pillH = max(h-4, 1)
	}
	xInset := min(float32(3), max(float32(0), (w-16)*0.25))
	yInset := max(float32(0), (h-pillH)*0.5)
	rad := min(float32(8), min(max(w-2*xInset, 0), pillH)*0.5)

	btn := ui.ButtonBase(c.Key(key)).
		DimDisabled(false).
		FocusRing(false).
		Size(w, h).
		Shrink(0).
		Radius(rad).
		AlignItems(ui.Center).
		Justify(ui.Center).
		Label(label)
	attachStateLayerWithTheme(th, btn, StateLayerOptions{
		Color: stateColor,
		Clip: func(r ui.Rect) (ui.Rect, [4]float32) {
			box := ui.Rect{
				X: r.X + xInset,
				Y: r.Y + yInset,
				W: max(0, r.W-2*xInset),
				H: max(0, r.H-2*yInset),
			}
			return box, [4]float32{rad, rad, rad, rad}
		},
	})
	btn.Children(func() {
		ui.Box(c).Size(12, 12).Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			draw(p, r, ink)
		})
	})
	return btn
}

func drawTitleBarMinimizeIcon(p *ui.Painter, r ui.Rect, ink ui.Color) {
	y := r.Y + r.H*0.5
	p.Line(r.X+1, y, r.X+r.W-1, y, 1.25, ink)
}

func drawTitleBarMaximizeIcon(p *ui.Painter, r ui.Rect, ink ui.Color, maximized bool) {
	if !maximized {
		p.Stroke(ui.Rect{X: r.X + 1.5, Y: r.Y + 1.5, W: r.W - 3, H: r.H - 3}, ink, 1, 1.25)
		return
	}
	// Restore glyph: front square + offset top-right back edge.
	s := r.W - 4.5
	front := ui.Rect{X: r.X + 1, Y: r.Y + 3.5, W: s, H: s}
	p.Stroke(front, ink, 1, 1.2)
	backX0 := front.X + 2.5
	backY0 := front.Y - 2.5
	backX1 := backX0 + s
	backY1 := backY0 + s
	p.Line(backX0, backY0, backX1, backY0, 1.2, ink)
	p.Line(backX1, backY0, backX1, backY1, 1.2, ink)
}

func drawTitleBarCloseIcon(p *ui.Painter, r ui.Rect, ink ui.Color) {
	p.Line(r.X+1.5, r.Y+1.5, r.X+r.W-1.5, r.Y+r.H-1.5, 1.35, ink)
	p.Line(r.X+r.W-1.5, r.Y+1.5, r.X+1.5, r.Y+r.H-1.5, 1.35, ink)
}

func drawTitleBarPinIcon(p *ui.Painter, r ui.Rect, ink ui.Color, pinned bool) {
	if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
		return
	}
	cx := r.X + r.W*0.5
	// Pin head, crossbar, and needle.
	head := ui.Rect{X: cx - 3.2, Y: r.Y + 2, W: 6.4, H: 5.5}
	if pinned {
		p.Fill(head, ink, 1.5)
	} else {
		p.Stroke(head, ink, 1.5, 1.3)
	}
	barY := r.Y + 8.5
	p.Line(cx-4.8, barY, cx+4.8, barY, 1.4, ink)
	p.Line(cx, barY, cx, r.Y+r.H-2, 1.4, ink)
}
