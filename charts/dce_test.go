package charts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZeroBloatDeadCodeElimination(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	goModBytes, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read repo go.mod: %v", err)
	}
	goSumBytes, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read repo go.sum: %v", err)
	}

	buildAndInspectNM := func(name, mainSrc string) string {
		t.Helper()
		dir := t.TempDir()

		modText := strings.Replace(string(goModBytes), "module github.com/imicola/mygo-md3e", "module chartsdcecheck", 1)
		modText += "\nrequire github.com/imicola/mygo-md3e v0.0.0\n"
		modText += "replace github.com/imicola/mygo-md3e => " + repoRoot + "\n"

		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modText), 0o600); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), goSumBytes, 0o600); err != nil {
			t.Fatalf("write go.sum: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o600); err != nil {
			t.Fatalf("write main.go: %v", err)
		}

		outBin := filepath.Join(dir, name+".bin")
		cmd := exec.Command("go", "build", "-mod=readonly", "-o", outBin, ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s failed: %v\n%s", name, err, string(out))
		}

		nmCmd := exec.Command("go", "tool", "nm", outBin)
		nmOut, err := nmCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go tool nm %s failed: %v\n%s", name, err, string(nmOut))
		}
		return string(nmOut)
	}

	// 1. Minimal app using root md3e.Button alone must NOT link charts/ or ui/chart.
	rootOnlyNM := buildAndInspectNM("root_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
)

func main() {
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			md3e.Button(c, "OK")
		})
	}, 100, 100)
	_ = tt
}
`)
	for _, forbidden := range []string{
		"github.com/imicola/mygo-md3e/charts",
		"github.com/egoist/mygo/ui/chart",
	} {
		if strings.Contains(rootOnlyNM, forbidden) {
			t.Fatalf("root-only Button binary linked forbidden package %q", forbidden)
		}
	}

	// 2. Binary using charts.AreaSparkline + charts.RingBuffer alone must NOT link
	// DonutChart, BarChart, symbols/, or media/.
	sparklineOnlyNM := buildAndInspectNM("sparkline_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/charts"
)

func main() {
	rb := charts.NewRingBuffer(8)
	rb.PushSlice(1, 2, 3, 4)
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			charts.AreaSparkline(c, charts.AreaSparklineOptions{Ring: rb})
		})
	}, 120, 60)
	_ = tt
}
`)
	for _, forbidden := range []string{
		"charts.DonutChart",
		"charts.BarChart",
		"github.com/imicola/mygo-md3e/symbols",
		"github.com/imicola/mygo-md3e/media",
	} {
		if strings.Contains(sparklineOnlyNM, forbidden) {
			t.Fatalf("AreaSparkline-only binary linked forbidden symbol/package %q", forbidden)
		}
	}
}
