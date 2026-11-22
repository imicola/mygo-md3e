package md3e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/imicola/mygo-md3e/shape"
)

func TestButtonDeadCodeElimination(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.go")
	if err := os.WriteFile(main, []byte(`package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e"
)

func main() {
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.Button(c, "OK")
	}, 100, 100)
	tt.Frame()
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command("go", args...)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOPROXY=off")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
		return out
	}
	build := func(name, source string) (string, int64) {
		t.Helper()
		binary := filepath.Join(dir, name)
		run("build", "-mod=readonly", "-buildvcs=false", "-o", binary, source)
		info, err := os.Stat(binary)
		if err != nil {
			t.Fatal(err)
		}
		return binary, info.Size()
	}

	button, buttonBytes := build("button.bin", main)
	nm := run("tool", "nm", button)
	const module = "github.com/imicola/mygo-md3e"
	if !bytes.Contains(nm, []byte(module+".Button\n")) {
		t.Fatal("the fixture must link md3e.Button")
	}
	forbidden := []string{
		module + "/symbols.", module + "/charts.", module + "/media.",
		module + "/docview.", module + "/codeview.",
		"github.com/egoist/mygo/ui/chart.", "github.com/egoist/mygo/ui/markdown.",
		"github.com/egoist/mygo/ui/editor.", "github.com/egoist/mygo/ui/netimage.",
		"github.com/egoist/mygo/ui/video.",
		module + "/shape.All", module + "/shape.ByName",
		module + ".LoadingIndicator", module + ".DataTable", module + ".Carousel",
	}
	for _, entry := range shape.All() {
		forbidden = append(forbidden, module+"/shape."+entry.Name)
	}
	for _, name := range forbidden {
		if bytes.Contains(nm, []byte(name)) {
			t.Errorf("Button binary links %q", name)
		}
	}

	_, galleryBytes := build("gallery.bin", "./examples/gallery")
	t.Logf("pure-Go binaries (%s/%s %s): Button %d bytes, Gallery %d bytes",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), buttonBytes, galleryBytes)
	// Keep absolute budgets on the platform and toolchain family used for
	// the baseline. Linker sections and frameworks differ on other targets.
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && strings.HasPrefix(runtime.Version(), "go1.27.") {
		if buttonBytes > 13_900_000 {
			t.Errorf("Button binary is %d bytes, want <= 13,900,000", buttonBytes)
		}
		if galleryBytes > 18_500_000 {
			t.Errorf("Gallery binary is %d bytes, want <= 18,500,000", galleryBytes)
		}
	}
}
