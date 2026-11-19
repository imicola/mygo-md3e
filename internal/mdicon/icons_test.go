package mdicon

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func inkBounds(img *image.RGBA, bg color.RGBA) (count int, minX, minY, maxX, maxY int) {
	b := img.Bounds()
	minX, minY = b.Max.X, b.Max.Y
	maxX, maxY = b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if px != bg {
				count++
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	return count, minX, minY, maxX, maxY
}

func TestBuiltinIconsRenderBoundsAndCentering(t *testing.T) {
	icons := []struct {
		name string
		ic   *Icon
	}{
		{"Check", Check},
		{"Minus", Minus},
		{"Close", Close},
		{"Clear", Clear},
		{"ChevronDown", ChevronDown},
		{"ChevronUp", ChevronUp},
		{"ChevronLeft", ChevronLeft},
		{"ChevronRight", ChevronRight},
		{"ArrowBack", ArrowBack},
		{"ArrowForward", ArrowForward},
		{"Add", Add},
		{"Edit", Edit},
		{"Search", Search},
		{"Menu", Menu},
		{"MoreVert", MoreVert},
		{"MoreHoriz", MoreHoriz},
		{"Visibility", Visibility},
		{"VisibilityOff", VisibilityOff},
		{"Calendar", Calendar},
		{"Schedule", Schedule},
		{"Clock", Clock},
		{"ErrorOutline", ErrorOutline},
		{"Star", Star},
	}

	bg := ui.RGB(255, 255, 255)
	fg := ui.RGB(20, 20, 20)
	bgRGBA := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for _, tc := range icons {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ic == nil {
				t.Fatalf("%s is nil", tc.name)
			}
			// Path() should parse cleanly and return distinct copies.
			p1 := tc.ic.Path()
			p2 := tc.ic.Path()
			if p1 == nil || p2 == nil || p1 == p2 {
				t.Fatalf("%s.Path() should allocate a fresh non-nil path", tc.name)
			}

			// Render on 24x24 standard canvas.
			img24 := ui.Render(func(c *ui.Context) {
				ui.Box(c).Size(24, 24).Background(bg).Draw(func(p *ui.Painter, r ui.Rect) {
					Draw(p, tc.ic, r, fg)
				})
			}, 24, 24, 1)

			n24, x0, y0, x1, y1 := inkBounds(img24, bgRGBA)
			if n24 < 12 {
				t.Fatalf("%s 24x24 inked pixel count = %d, want >= 12", tc.name, n24)
			}
			// Must stay inside [0, 23] and roughly centered within [0..24].
			if x0 < 0 || y0 < 0 || x1 > 23 || y1 > 23 {
				t.Fatalf("%s 24x24 bounds [%d,%d]..[%d,%d] out of canvas", tc.name, x0, y0, x1, y1)
			}
			cx := float64(x0+x1) * 0.5
			cy := float64(y0+y1) * 0.5
			if cx < 8.5 || cx > 14.5 || cy < 8.5 || cy > 14.5 {
				t.Fatalf("%s 24x24 bounds [%d,%d]..[%d,%d] not centered (center=(%.1f, %.1f))",
					tc.name, x0, y0, x1, y1, cx, cy)
			}

			// Render on compact 16x16 and non-square offset 48x32 rectangles.
			img16 := ui.Render(func(c *ui.Context) {
				ui.Box(c).Size(16, 16).Background(bg).Draw(func(p *ui.Painter, r ui.Rect) {
					tc.ic.Draw(p, r, fg)
				})
			}, 16, 16, 1)
			n16, _, _, _, _ := inkBounds(img16, bgRGBA)
			if n16 < 6 {
				t.Fatalf("%s 16x16 inked pixel count = %d, want >= 6", tc.name, n16)
			}

			// Render in Stroke mode on offset 48x32 box centered at (24, 16).
			imgStroke := ui.Render(func(c *ui.Context) {
				ui.Box(c).Size(48, 32).Background(bg).Draw(func(p *ui.Painter, r ui.Rect) {
					tc.ic.Stroke(p, r, 1.5, fg)
				})
			}, 48, 32, 1)
			ns, sx0, _, sx1, _ := inkBounds(imgStroke, bgRGBA)
			if ns < 12 || sx0 < 6 || sx1 > 41 {
				t.Fatalf("%s stroke render unexpected: count=%d x=[%d..%d]", tc.name, ns, sx0, sx1)
			}
		})
	}
}

func TestCutoutIconsPreserveInteriorHoles(t *testing.T) {
	bg := ui.RGB(255, 255, 255)
	fg := ui.RGB(0, 0, 0)
	bgRGBA := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	render24 := func(ic *Icon) *image.RGBA {
		return ui.Render(func(c *ui.Context) {
			ui.Box(c).Size(24, 24).Background(bg).Draw(func(p *ui.Painter, r ui.Rect) {
				ic.Draw(p, r, fg)
			})
		}, 24, 24, 1)
	}

	// Search magnifying glass center (9, 9) must remain hollow white.
	if px := render24(Search).RGBAAt(9, 9); px != bgRGBA {
		t.Fatalf("Search lens interior (9,9) = %+v, want hollow white %+v", px, bgRGBA)
	}
	// Calendar body (12, 14) must remain hollow white.
	if px := render24(Calendar).RGBAAt(12, 14); px != bgRGBA {
		t.Fatalf("Calendar body interior (12,14) = %+v, want hollow white %+v", px, bgRGBA)
	}
	// Schedule clock face hollow region (8, 12) must remain hollow white.
	if px := render24(Schedule).RGBAAt(8, 12); px != bgRGBA {
		t.Fatalf("Schedule clock face (8,12) = %+v, want hollow white %+v", px, bgRGBA)
	}
	// Visibility must fill pupil center (12, 12) while leaving iris gap (9, 12) white.
	vis := render24(Visibility)
	if px := vis.RGBAAt(12, 12); px == bgRGBA {
		t.Fatal("Visibility pupil (12,12) should be inked")
	}
	if px := vis.RGBAAt(8, 12); px != bgRGBA {
		t.Fatalf("Visibility iris ring (8,12) = %+v, want hollow white %+v", px, bgRGBA)
	}
	// Clear badge center (12, 12) has white X cutout over black circle.
	clr := render24(Clear)
	if px := clr.RGBAAt(12, 12); px != bgRGBA {
		t.Fatalf("Clear badge X center (12,12) = %+v, want hollow white %+v", px, bgRGBA)
	}
	if px := clr.RGBAAt(12, 6); px == bgRGBA {
		t.Fatal("Clear badge solid ring at (12,6) should be inked")
	}
}

func TestZeroBloatNoInitAndNoGlobalRegistry(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		// Use explicit file parse fallback since fs.FileInfo matches cleanly.
		t.Fatalf("ParseDir unexpected error: %v", err)
	}
	pkg := pkgs["mdicon"]
	if pkg == nil {
		t.Fatal("mdicon package not found")
	}
	for filename, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == "init" {
					t.Fatalf("%s must not declare init()", filepath.Base(filename))
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					vs := spec.(*ast.ValueSpec)
					for _, val := range vs.Values {
						switch val.(type) {
						case *ast.CompositeLit:
							// Ensure no package-level slice/array/map literal holding icons.
							if lit, ok := val.(*ast.CompositeLit); ok {
								switch lit.Type.(type) {
								case *ast.ArrayType, *ast.MapType:
									t.Fatalf("%s must not pack icons into a package-level slice/map %s",
										filepath.Base(filename), vs.Names[0].Name)
								}
							}
						}
					}
				}
			}
		}
	}
}
