package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

type visualBoardSpec struct {
	name         string
	width        int
	height       int
	dark         bool
	contrast     tokens.Contrast
	surfaceStyle tokens.SurfaceStyle
	rtl          bool
}

var monitorVisualBoardSpecs = []visualBoardSpec{
	{name: "monitor_dark_compact_520", width: 520, height: 760, dark: true, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceTonal},
	{name: "monitor_dark_medium_760", width: 760, height: 760, dark: true, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceTonal},
	{name: "monitor_dark_expanded_1280", width: 1280, height: 760, dark: true, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceTonal},
	{name: "monitor_light_expanded_1280", width: 1280, height: 760, dark: false, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceTonal},
	{name: "monitor_hc_dark_1280", width: 1280, height: 760, dark: true, contrast: tokens.ContrastHigh, surfaceStyle: tokens.SurfaceTonal},
	{name: "monitor_neutral_dark_expanded_1280", width: 1280, height: 760, dark: true, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceNeutral},
	{name: "monitor_neutral_light_expanded_1280", width: 1280, height: 760, dark: false, contrast: tokens.ContrastStandard, surfaceStyle: tokens.SurfaceNeutral},
}

func resolveVisualBoardOutputDir(t *testing.T) (dir string, keep bool) {
	t.Helper()
	if explicit := strings.TrimSpace(os.Getenv("MD3E_EXPORT_BOARD_DIR")); explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			t.Fatalf("resolve MD3E_EXPORT_BOARD_DIR: %v", err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			t.Fatalf("create MD3E_EXPORT_BOARD_DIR %q: %v", abs, err)
		}
		return abs, true
	}
	if strings.TrimSpace(os.Getenv("MD3E_EXPORT_BOARD")) == "1" {
		tmp, err := os.MkdirTemp("", "md3e-visual-board-*")
		if err != nil {
			t.Fatalf("create visual board temp dir: %v", err)
		}
		return tmp, true
	}
	return t.TempDir(), false
}

// TestMonitorSliceVisualBoardExport renders the deterministic Monitor slice at
// Compact (520), Medium (760), and Expanded (1280) widths across Dark, Light,
// and HighContrast themes via headless ui.Tester, exporting PNGs to a temporary
// directory (or MD3E_EXPORT_BOARD_DIR when set) without ever writing into the
// repository tree.
func TestMonitorSliceVisualBoardExport(t *testing.T) {
	outDir, keep := resolveVisualBoardOutputDir(t)
	lumas := make(map[string]float64, len(monitorVisualBoardSpecs))

	for _, spec := range monitorVisualBoardSpecs {
		opts := defaultMonitorSliceOptions()
		opts.Dark = spec.dark
		opts.Contrast = spec.contrast
		opts.SurfaceStyle = spec.surfaceStyle
		opts.RTL = spec.rtl

		f := newMonitorSliceFixture(opts)
		t.Cleanup(f.app.media.close)

		tt := ui.NewTester(f.view, spec.width, spec.height)
		tt.Frame()
		if tt.FramePending() {
			t.Fatalf("%s: settled ReduceMotion visual board still has pending frames", spec.name)
		}
		img := tt.Image()
		if img.Bounds().Dx() != spec.width || img.Bounds().Dy() != spec.height {
			t.Fatalf("%s: image size = %dx%d, want %dx%d",
				spec.name, img.Bounds().Dx(), img.Bounds().Dy(), spec.width, spec.height)
		}

		dst := filepath.Join(outDir, spec.name+".png")
		file, err := os.Create(dst)
		if err != nil {
			t.Fatalf("%s: create %q: %v", spec.name, dst, err)
		}
		if err := png.Encode(file, img); err != nil {
			file.Close()
			t.Fatalf("%s: encode PNG: %v", spec.name, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("%s: close PNG: %v", spec.name, err)
		}

		// Round-trip decode verifies every exported screenshot is a valid PNG.
		rf, err := os.Open(dst)
		if err != nil {
			t.Fatalf("%s: reopen %q: %v", spec.name, dst, err)
		}
		decoded, err := png.DecodeConfig(rf)
		rf.Close()
		if err != nil {
			t.Fatalf("%s: decode PNG config: %v", spec.name, err)
		}
		if decoded.Width != spec.width || decoded.Height != spec.height {
			t.Fatalf("%s: decoded PNG size = %dx%d, want %dx%d",
				spec.name, decoded.Width, decoded.Height, spec.width, spec.height)
		}
		lumas[spec.name] = meanLuma(img)
	}

	if lumas["monitor_light_expanded_1280"] <= lumas["monitor_dark_expanded_1280"]+40 {
		t.Fatalf("light board luma (%.1f) not clearly brighter than dark board (%.1f)",
			lumas["monitor_light_expanded_1280"], lumas["monitor_dark_expanded_1280"])
	}
	if keep {
		t.Logf("exported %d Monitor visual state boards to %s (%s)",
			len(monitorVisualBoardSpecs), outDir, fmt.Sprintf("dark=%.1f light=%.1f",
				lumas["monitor_dark_expanded_1280"], lumas["monitor_light_expanded_1280"]))
	}
}
