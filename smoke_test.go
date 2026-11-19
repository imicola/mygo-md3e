package md3e_test

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-md3e"
)

// TestSmokeHeadlessFrame builds the root module's view in a tester, draws it
// without a window, sends a click, and checks that the view settles: no
// animation frames are left pending.
func TestSmokeHeadlessFrame(t *testing.T) {
	clicks := 0
	view := func(c *ui.Context) {
		ui.Text(c, "mygo-md3e "+md3e.Version)
		if ui.Button(c, "Increment").Clicked() {
			clicks++
		}
	}

	tt := ui.NewTester(view, 320, 240)
	if !tt.HasText("mygo-md3e " + md3e.Version) {
		t.Fatalf("the view's text is missing: %q", tt.Texts())
	}
	if err := tt.Click("Increment"); err != nil {
		t.Fatalf("click the button: %v", err)
	}
	if clicks != 1 {
		t.Errorf("the button was clicked %d times, want 1", clicks)
	}
	if tt.FramePending() {
		t.Error("the view keeps asking for frames after the click")
	}

	img := tt.Image()
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 240 {
		t.Fatalf("frame is %v, want 320x240", img.Bounds())
	}
	if isBlank(img) {
		t.Error("the frame drew nothing")
	}
}

// TestSmokeRender checks the one-shot headless renderer the component tests
// will use for pixel assertions.
func TestSmokeRender(t *testing.T) {
	img := ui.Render(func(c *ui.Context) {
		ui.Text(c, "render")
	}, 160, 120, 1)
	if img.Bounds().Dx() != 160 || img.Bounds().Dy() != 120 {
		t.Fatalf("render is %v, want 160x120", img.Bounds())
	}
	if isBlank(img) {
		t.Error("the render drew nothing")
	}
}

// isBlank reports whether every pixel of the image is the same color.
func isBlank(img *image.RGBA) bool {
	first := img.RGBAAt(0, 0)
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y) != first {
				return false
			}
		}
	}
	return true
}
