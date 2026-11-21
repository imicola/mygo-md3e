package symbols

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Compile-time check that (*Symbol).Draw matches md3e.IconDraw's exact signature.
var _ func(p *ui.Painter, dst ui.Rect, c ui.Color) = Home.Draw

type parsedSymbolSpec struct {
	file    string
	name    string
	d       string
	evenOdd bool
}

func collectExportedSymbols(t *testing.T) []parsedSymbolSpec {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []parsedSymbolSpec
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		// Enforce zero-bloat & zero-cycle structural invariants.
		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if p == "github.com/imicola/mygo-md3e" {
				t.Fatalf("%s must not import root md3e package", name)
			}
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == "init" {
					t.Fatalf("%s defines forbidden init() function", name)
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					vs := spec.(*ast.ValueSpec)
					for i, ident := range vs.Names {
						if !ident.IsExported() {
							continue
						}
						if i >= len(vs.Values) {
							t.Fatalf("%s: exported var %s has no initializer", name, ident.Name)
						}
						unary, ok := vs.Values[i].(*ast.UnaryExpr)
						if !ok || unary.Op != token.AND {
							t.Fatalf("%s: exported var %s must be &Symbol{...} (no registry/map/slice/OnceValue)", name, ident.Name)
						}
						cl, ok := unary.X.(*ast.CompositeLit)
						if !ok {
							t.Fatalf("%s: exported var %s must be &Symbol{...}", name, ident.Name)
						}
						id, ok := cl.Type.(*ast.Ident)
						if !ok || id.Name != "Symbol" {
							t.Fatalf("%s: exported var %s must have type *Symbol", name, ident.Name)
						}
						var pathStr string
						var evenOdd bool
						for _, elt := range cl.Elts {
							kv, ok := elt.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							k := kv.Key.(*ast.Ident).Name
							switch k {
							case "d":
								lit := kv.Value.(*ast.BasicLit)
								pathStr, err = strconv.Unquote(lit.Value)
								if err != nil {
									t.Fatalf("%s:%s unquote d: %v", name, ident.Name, err)
								}
							case "evenOdd":
								if b, ok := kv.Value.(*ast.Ident); ok && b.Name == "true" {
									evenOdd = true
								}
							}
						}
						out = append(out, parsedSymbolSpec{
							file:    name,
							name:    ident.Name,
							d:       pathStr,
							evenOdd: evenOdd,
						})
					}
				}
			}
		}
	}
	return out
}

func TestAllExportedSymbolsCountParseAndRender(t *testing.T) {
	specs := collectExportedSymbols(t)
	if len(specs) < 200 {
		t.Fatalf("exported symbol count = %d, want >= 200", len(specs))
	}

	seen := make(map[string]string, len(specs))
	for _, s := range specs {
		if prev, dup := seen[s.name]; dup {
			t.Fatalf("duplicate symbol %s in %s and %s", s.name, prev, s.file)
		}
		seen[s.name] = s.file
		if s.d == "" {
			t.Fatalf("%s.%s has empty path", s.file, s.name)
		}
		p, err := ui.ParsePath(s.d)
		if err != nil {
			t.Fatalf("%s.%s: ui.ParsePath failed: %v", s.file, s.name, err)
		}
		if p == nil {
			t.Fatalf("%s.%s: ui.ParsePath returned nil", s.file, s.name)
		}
	}

	// Render every exported symbol into a 24x24 slot on a batched grid canvas
	// to verify every single path paints visible pixels inside its slot.
	const (
		cols   = 16
		cell   = 32
		margin = 4
		iconSz = 24
	)
	rows := (len(specs) + cols - 1) / cols
	w := cols * cell
	h := rows * cell

	syms := make([]*Symbol, len(specs))
	for i, sp := range specs {
		if sp.evenOdd {
			syms[i] = CustomEvenOdd(sp.d)
		} else {
			syms[i] = Custom(sp.d)
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(float32(w), float32(h)).Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
			ink := ui.RGB(20, 20, 20)
			for i, sym := range syms {
				col := i % cols
				row := i / cols
				dst := ui.Rect{
					X: r.X + float32(col*cell+margin),
					Y: r.Y + float32(row*cell+margin),
					W: iconSz,
					H: iconSz,
				}
				sym.Draw(p, dst, ink)
			}
		})
	}, w, h)

	img := tt.Image()
	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for i, sp := range specs {
		col := i % cols
		row := i / cols
		x0 := col*cell + margin
		y0 := row*cell + margin
		painted := 0
		for y := y0; y < y0+iconSz; y++ {
			for x := x0; x < x0+iconSz; x++ {
				px := img.RGBAAt(x, y)
				if px != bg {
					painted++
				}
			}
		}
		if painted < 12 {
			t.Fatalf("%s.%s painted only %d pixels inside its 24x24 slot", sp.file, sp.name, painted)
		}
		// Check 1px outer gutter around the cell does not receive runaway strokes.
		bleed := 0
		for x := col * cell; x < (col+1)*cell; x++ {
			if img.RGBAAt(x, row*cell) != bg || img.RGBAAt(x, (row+1)*cell-1) != bg {
				bleed++
			}
		}
		if bleed > 0 {
			t.Fatalf("%s.%s bled %d pixels outside 24x24 viewBox gutter", sp.file, sp.name, bleed)
		}
	}
}

func TestSymbolHelpersAndEdgeCases(t *testing.T) {
	var nilSym *Symbol
	if nilSym.Path() != nil {
		t.Fatal("nil Symbol.Path() must return nil")
	}
	nilSym.Draw(nil, ui.Rect{W: 24, H: 24}, ui.RGB(0, 0, 0))
	nilSym.Stroke(nil, ui.Rect{W: 24, H: 24}, 2, ui.RGB(0, 0, 0))

	empty := Custom("")
	if empty.Path() != nil {
		t.Fatal("empty Custom Symbol.Path() must return nil")
	}

	p1 := Home.Path()
	p2 := Home.Path()
	if p1 == nil || p2 == nil || p1 == p2 {
		t.Fatal("Path() must return a fresh *ui.Path on every call")
	}

	custom := Custom("M4 12L12 4L20 12L12 20Z")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(64, 32).Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
			custom.Draw(p, ui.Rect{X: 4, Y: 4, W: 24, H: 24}, ui.RGB(0, 0, 0))
			custom.Stroke(p, ui.Rect{X: 36, Y: 4, W: 24, H: 24}, 2, ui.RGB(0, 0, 0))
		})
	}, 64, 32)
	img := tt.Image()
	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if img.RGBAAt(16, 16) == bg {
		t.Fatal("Custom.Draw did not fill diamond center")
	}
	if img.RGBAAt(48, 8) == bg {
		t.Fatal("Custom.Stroke did not paint diamond outline")
	}
}

// TestDeadCodeElimination builds two minimal pure-Go binaries in t.TempDir()
// and verifies by scanning compiled binary bytes that unreferenced icons from
// every category file are completely eliminated by the Go linker.
func TestDeadCodeElimination(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("abs repo root: %v", err)
	}
	rootGoMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read root go.mod: %v", err)
	}
	rootGoSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read root go.sum: %v", err)
	}

	// Derive a self-contained offline go.mod from root go.mod so transitive
	// indirect dependencies resolve cleanly from the local Go module cache.
	var modLines []string
	for _, line := range strings.Split(string(rootGoMod), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "module ") {
			modLines = append(modLines, "module dcecheck")
			continue
		}
		if strings.HasPrefix(trim, "tool ") {
			continue
		}
		modLines = append(modLines, line)
	}
	baseGoMod := strings.Join(modLines, "\n") + fmt.Sprintf("\nrequire github.com/imicola/mygo-md3e v0.0.0\nreplace github.com/imicola/mygo-md3e => %q\n", filepath.ToSlash(repoRoot))

	buildAndRead := func(name, src string) []byte {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(baseGoMod), 0o600); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), rootGoSum, 0o600); err != nil {
			t.Fatalf("write go.sum: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		outBin := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-mod=readonly", "-o", outBin, ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s failed: %v\n%s", name, err, string(out))
		}
		data, err := os.ReadFile(outBin)
		if err != nil {
			t.Fatalf("read binary %s: %v", name, err)
		}
		return data
	}

	binHomeOnly := buildAndRead("home_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e/symbols"
)

func main() {
	symbols.Home.Draw(nil, ui.Rect{}, ui.Color{})
}
`)

	binHomeAndTerminal := buildAndRead("home_and_terminal", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e/symbols"
)

func main() {
	symbols.Home.Draw(nil, ui.Rect{}, ui.Color{})
	symbols.Terminal.Draw(nil, ui.Rect{}, ui.Color{})
}
`)

	// Positive assertion: linked symbol path string MUST appear in the binary.
	if !bytes.Contains(binHomeOnly, []byte(Home.d)) {
		t.Fatal("home_only binary did not contain referenced Home SVG path string (detector check failed)")
	}

	// Negative assertions: unreferenced symbols from EVERY category file must
	// be completely absent from home_only.
	unreferenced := []struct {
		name string
		sym  *Symbol
	}{
		{"MenuOpen (navigation)", MenuOpen},
		{"Search (actions)", Search},
		{"PlayArrow (media)", PlayArrow},
		{"FormatBold (editor)", FormatBold},
		{"FolderZip (files)", FolderZip},
		{"Terminal (system)", Terminal},
		{"Settings (system)", Settings},
		{"CheckCircle (status)", CheckCircle},
		{"WorkspacePremium (status)", WorkspacePremium},
	}
	for _, u := range unreferenced {
		if bytes.Contains(binHomeOnly, []byte(u.sym.d)) {
			t.Fatalf("DCE failure: unreferenced symbol %s was linked into home_only binary", u.name)
		}
	}

	// Positive + negative assertions on binary B.
	if !bytes.Contains(binHomeAndTerminal, []byte(Home.d)) {
		t.Fatal("home_and_terminal binary missing Home")
	}
	if !bytes.Contains(binHomeAndTerminal, []byte(Terminal.d)) {
		t.Fatal("home_and_terminal binary missing Terminal")
	}
	if bytes.Contains(binHomeAndTerminal, []byte(WorkspacePremium.d)) {
		t.Fatal("DCE failure: unreferenced WorkspacePremium leaked into home_and_terminal binary")
	}
}
