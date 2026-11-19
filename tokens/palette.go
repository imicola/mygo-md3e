package tokens

import (
	"math"
	"sync"

	"github.com/egoist/mygo/ui"
)

// TonalPalette is a hue and chroma of HCT with the colors at every tone.
// Colors from one palette share their hue and chroma and differ only in tone,
// so a palette is the building block of a dynamic color scheme.
type TonalPalette struct {
	hue    float64
	chroma float64

	mu    sync.RWMutex
	tones map[float64]ui.Color
}

// NewTonalPalette returns the palette of hue and chroma. A hue outside 0..360
// wraps around and a negative chroma is zero.
func NewTonalPalette(hue, chroma float64) *TonalPalette {
	return &TonalPalette{
		hue:    SanitizeDegrees(hue),
		chroma: math.Max(0, chroma),
		tones:  make(map[float64]ui.Color),
	}
}

// PaletteFromColor returns the palette of c's hue and chroma.
func PaletteFromColor(c ui.Color) *TonalPalette {
	hct := HCTFromColor(c)
	return NewTonalPalette(hct.Hue, hct.Chroma)
}

// Hue returns the palette's hue in degrees.
func (p *TonalPalette) Hue() float64 { return p.hue }

// Chroma returns the palette's chroma.
func (p *TonalPalette) Chroma() float64 { return p.chroma }

// HCT returns the palette's color at tone.
func (p *TonalPalette) HCT(tone float64) HCT {
	return HCT{Hue: p.hue, Chroma: p.chroma, Tone: min(max(tone, 0), 100)}
}

// Tone returns the palette's opaque color at tone, caching what it computes.
func (p *TonalPalette) Tone(tone float64) ui.Color {
	tone = min(max(tone, 0), 100)
	p.mu.RLock()
	c, ok := p.tones[tone]
	p.mu.RUnlock()
	if ok {
		return c
	}
	c = p.HCT(tone).ToColor()
	p.mu.Lock()
	p.tones[tone] = c
	p.mu.Unlock()
	return c
}
