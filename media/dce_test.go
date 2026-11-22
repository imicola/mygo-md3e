package media_test

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

		// Preserve all direct & indirect requires from root go.mod so -mod=readonly matches go.sum offline.
		modText := strings.Replace(string(goModBytes), "module github.com/imicola/mygo-md3e", "module dcecheck", 1)
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

	// 1. Minimal app using md3e.Button + md3e.Image must NOT link media/, ui/netimage, ui/video, or symbols/.
	rootOnlyNM := buildAndInspectNM("root_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
)

func main() {
	var tt *ui.Tester
	tt = ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			md3e.Button(c, "OK")
			md3e.Image(c, nil, md3e.ImageOptions{Width: 32, Height: 32})
		})
	}, 100, 100)
	_ = tt
}
`)
	for _, forbidden := range []string{
		"github.com/imicola/mygo-md3e/media",
		"github.com/egoist/mygo/ui/netimage",
		"github.com/egoist/mygo/ui/video",
		"github.com/imicola/mygo-md3e/symbols",
	} {
		if strings.Contains(rootOnlyNM, forbidden) {
			t.Fatalf("root-only Button+Image binary linked forbidden package %q", forbidden)
		}
	}

	// 2. Binary using media.NetImage alone must NOT link ui/video or tokens.SeedFromImage.
	netImageOnlyNM := buildAndInspectNM("netimage_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/media"
)

func main() {
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			media.NetImage(c, "", media.NetImageOptions{Width: 40, Height: 40})
		})
	}, 100, 100)
	_ = tt
}
`)
	for _, forbidden := range []string{
		"github.com/egoist/mygo/ui/video",
		"tokens.SeedFromImage",
	} {
		if strings.Contains(netImageOnlyNM, forbidden) {
			t.Fatalf("media.NetImage-only binary linked forbidden symbol %q", forbidden)
		}
	}
}
