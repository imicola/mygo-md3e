package docview_test

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

		modText := strings.Replace(string(goModBytes), "module github.com/imicola/mygo-md3e", "module docviewdcecheck", 1)
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

	// 1. Minimal app using root md3e.Button alone must NOT link docview/, codeview/, ui/markdown, or ui/editor.
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
		"github.com/imicola/mygo-md3e/docview",
		"github.com/imicola/mygo-md3e/codeview",
		"github.com/egoist/mygo/ui/markdown",
		"github.com/egoist/mygo/ui/editor",
	} {
		if strings.Contains(rootOnlyNM, forbidden) {
			t.Fatalf("root-only Button binary linked forbidden package %q", forbidden)
		}
	}

	// 2. Binary using docview.Markdown alone must NOT link editor.View, editor.DiffView, media/, or charts/.
	markdownOnlyNM := buildAndInspectNM("markdown_only", `package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
)

func main() {
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			docview.Markdown(c, "# Hello\n\nWorld")
		})
	}, 240, 180)
	_ = tt
}
`)
	for _, forbidden := range []string{
		"github.com/egoist/mygo/ui/editor.View",
		"github.com/egoist/mygo/ui/editor.DiffView",
		"github.com/imicola/mygo-md3e/media",
		"github.com/imicola/mygo-md3e/charts",
	} {
		if strings.Contains(markdownOnlyNM, forbidden) {
			t.Fatalf("docview.Markdown-only binary linked forbidden symbol/package %q", forbidden)
		}
	}
}
