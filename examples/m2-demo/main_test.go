package main

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestM2DemoInteractionsAndSettling(t *testing.T) {
	d := newDemoState()
	tt := ui.NewTester(d.view, 1240, 820)

	if !tt.HasText("MyGo MD3E — M1 & M2 Native Preview") {
		t.Fatalf("missing demo title in %q", tt.Texts())
	}
	if isBlank(tt.Image()) {
		t.Fatal("initial frame is blank")
	}

	// Click a shape tile to trigger a spring morph.
	if err := tt.Click("Morph to Heart"); err != nil {
		t.Fatalf("click Morph to Heart: %v", err)
	}
	if d.currentShapeIdx() != d.nameIndex["Heart"] {
		t.Fatalf("current shape = %s, want Heart", d.shapes[d.currentShapeIdx()].Name)
	}
	// Mid-flight frame should be pending, then settle completely.
	tt.Advance(100 * time.Millisecond)
	tt.Advance(2 * time.Second)
	if tt.FramePending() {
		t.Fatal("morph animation did not settle after 2.1s")
	}

	// Switch to Dark + Standard motion + Seed Blue.
	if err := tt.Click("Dark"); err != nil {
		t.Fatalf("click Dark: %v", err)
	}
	if err := tt.Click("Standard"); err != nil {
		t.Fatalf("click Standard: %v", err)
	}
	if err := tt.Click("Seed Blue"); err != nil {
		t.Fatalf("click Seed Blue: %v", err)
	}
	if d.darkIdx != 1 || d.motionSchemeIdx != 1 || d.seedIdx != 1 {
		t.Fatalf("unexpected state after clicks: dark=%d motion=%d seed=%d",
			d.darkIdx, d.motionSchemeIdx, d.seedIdx)
	}

	// Verify ReduceMotion snaps without leaving frames pending.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("Next Pair"); err != nil {
		t.Fatalf("click Next Pair: %v", err)
	}
	tt.Advance(16 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion left animation frames pending")
	}
}

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
