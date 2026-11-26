package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/media"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

const (
	audioTickInterval = 250 * time.Millisecond

	maskModeExpressive = 0
	maskModeRounded    = 1
	maskModeMono       = 2

	toneFilterAll  = 0
	toneFilterWarm = 1
	toneFilterCool = 2
)

type artRecipe struct {
	id        string
	title     string
	artist    string
	album     string
	duration  time.Duration
	shapeName string
	shape     *shape.RoundedPolygon
	warm      bool
	bgTop     color.NRGBA
	bgBottom  color.NRGBA
	accentA   color.NRGBA
	accentB   color.NRGBA
	glow      color.NRGBA
	style     int
}

type mediaArtwork struct {
	id        string
	title     string
	artist    string
	album     string
	duration  time.Duration
	shapeName string
	shape     *shape.RoundedPolygon
	warm      bool
	matchKey  string

	raw     *image.RGBA
	bitmap  *ui.Bitmap
	seed    ui.Color
	seedHex string
	palette []ui.Color
}

type mediaClip struct {
	id       string
	title    string
	subtitle string
	player   *ui.VideoPlayer
	err      error
}

type mediaState struct {
	artworks []mediaArtwork
	clips    []mediaClip

	// Track & queue state.
	activeTrack   int
	audioPlaying  bool
	shuffle       bool
	repeat        bool
	audioPos      time.Duration
	audioBuffered float64
	lastAudioTick time.Time
	queueFilter   string
	showQueue     bool
	compactQueue  bool

	// Cover dynamic color extraction state (tokens.SeedFromImage / PaletteFromImage).
	selectedArtIdx    int
	followCoverTheme  bool
	customPaletteSeed ui.Color

	// Video stage state.
	activeClip      int
	videoFullscreen bool

	// Expressive gallery & lightbox state.
	galleryMaskMode     int
	galleryToneFilter   int
	previewPlaceholders bool
	placeholderErrFixed bool
	lightboxOpen        bool
	lightboxIdx         int
	lightboxOpenIdx     int
	lightboxItems       []media.LightboxItem
}

func newMediaState() *mediaState {
	recipes := []artRecipe{
		{
			id: "solar", title: "Solar Flare", artist: "Helios Synth", album: "Horizon Drift",
			duration: 204 * time.Second, shapeName: "Cookie9Sided", shape: shape.Cookie9Sided(), warm: true,
			bgTop:    color.NRGBA{R: 54, G: 22, B: 18, A: 255},
			bgBottom: color.NRGBA{R: 224, G: 98, B: 26, A: 255},
			accentA:  color.NRGBA{R: 255, G: 188, B: 66, A: 255},
			accentB:  color.NRGBA{R: 244, G: 81, B: 60, A: 255},
			glow:     color.NRGBA{R: 255, G: 230, B: 140, A: 255},
			style:    0,
		},
		{
			id: "abyss", title: "Abyssal Tide", artist: "Nautilus Echo", album: "Pelagic",
			duration: 252 * time.Second, shapeName: "SoftBurst", shape: shape.SoftBurst(), warm: false,
			bgTop:    color.NRGBA{R: 8, G: 30, B: 56, A: 255},
			bgBottom: color.NRGBA{R: 12, G: 124, B: 148, A: 255},
			accentA:  color.NRGBA{R: 48, G: 224, B: 208, A: 255},
			accentB:  color.NRGBA{R: 24, G: 140, B: 228, A: 255},
			glow:     color.NRGBA{R: 170, G: 252, B: 245, A: 255},
			style:    1,
		},
		{
			id: "sakura", title: "Sakura Bloom", artist: "Kyoto Tape", album: "Spring Static",
			duration: 176 * time.Second, shapeName: "Flower", shape: shape.Flower(), warm: true,
			bgTop:    color.NRGBA{R: 64, G: 20, B: 62, A: 255},
			bgBottom: color.NRGBA{R: 214, G: 64, B: 138, A: 255},
			accentA:  color.NRGBA{R: 255, G: 158, B: 204, A: 255},
			accentB:  color.NRGBA{R: 162, G: 84, B: 236, A: 255},
			glow:     color.NRGBA{R: 255, G: 224, B: 240, A: 255},
			style:    2,
		},
		{
			id: "canopy", title: "Emerald Canopy", artist: "Moss & Wire", album: "Fern Field",
			duration: 228 * time.Second, shapeName: "Clover4Leaf", shape: shape.Clover4Leaf(), warm: false,
			bgTop:    color.NRGBA{R: 14, G: 44, B: 32, A: 255},
			bgBottom: color.NRGBA{R: 38, G: 152, B: 96, A: 255},
			accentA:  color.NRGBA{R: 168, G: 232, B: 84, A: 255},
			accentB:  color.NRGBA{R: 28, G: 184, B: 132, A: 255},
			glow:     color.NRGBA{R: 228, G: 252, B: 182, A: 255},
			style:    1,
		},
		{
			id: "cobalt", title: "Cobalt Pulse", artist: "Vector Nine", album: "Midnight Grid",
			duration: 245 * time.Second, shapeName: "PuffyDiamond", shape: shape.PuffyDiamond(), warm: false,
			bgTop:    color.NRGBA{R: 18, G: 24, B: 74, A: 255},
			bgBottom: color.NRGBA{R: 54, G: 92, B: 230, A: 255},
			accentA:  color.NRGBA{R: 112, G: 178, B: 255, A: 255},
			accentB:  color.NRGBA{R: 132, G: 86, B: 246, A: 255},
			glow:     color.NRGBA{R: 208, G: 232, B: 255, A: 255},
			style:    2,
		},
		{
			id: "crimson", title: "Crimson Dusk", artist: "Amber Horizon", album: "Velvet Signal",
			duration: 195 * time.Second, shapeName: "Sunny", shape: shape.Sunny(), warm: true,
			bgTop:    color.NRGBA{R: 62, G: 14, B: 28, A: 255},
			bgBottom: color.NRGBA{R: 204, G: 48, B: 68, A: 255},
			accentA:  color.NRGBA{R: 255, G: 132, B: 94, A: 255},
			accentB:  color.NRGBA{R: 246, G: 186, B: 72, A: 255},
			glow:     color.NRGBA{R: 255, G: 214, B: 184, A: 255},
			style:    0,
		},
	}

	artworks := make([]mediaArtwork, len(recipes))
	lbItems := make([]media.LightboxItem, len(recipes))
	for i, r := range recipes {
		raw := synthesizeArtwork(192, 192, r)
		bmp := ui.NewBitmap(raw)
		seed := tokens.SeedFromImage(raw)
		pal := tokens.PaletteFromImage(raw, tokens.ExtractOptions{Desired: 4})
		hex := mediaColorHex(seed)
		matchKey := "workbench-gallery-" + r.id

		artworks[i] = mediaArtwork{
			id:        r.id,
			title:     r.title,
			artist:    r.artist,
			album:     r.album,
			duration:  r.duration,
			shapeName: r.shapeName,
			shape:     r.shape,
			warm:      r.warm,
			matchKey:  matchKey,
			raw:       raw,
			bitmap:    bmp,
			seed:      seed,
			seedHex:   hex,
			palette:   pal,
		}
		lbItems[i] = media.LightboxItem{
			Image:   bmp,
			Title:   r.title + " — " + r.artist,
			Caption: fmt.Sprintf("%s · %s mask · Dynamic seed %s", r.album, r.shapeName, hex),
			Label:   r.title,
		}
	}

	clips := []mediaClip{
		buildProceduralClip("amber-wave", "Amber Wave", "Solar spectrum · 2.4s loop",
			color.NRGBA{R: 26, G: 16, B: 14, A: 255},
			color.NRGBA{R: 235, G: 118, B: 32, A: 255},
			color.NRGBA{R: 255, G: 204, B: 96, A: 255}, 0),
		buildProceduralClip("prism-orbit", "Prism Orbit", "Orbital rings · 2.4s loop",
			color.NRGBA{R: 18, G: 16, B: 34, A: 255},
			color.NRGBA{R: 148, G: 102, B: 248, A: 255},
			color.NRGBA{R: 72, G: 214, B: 232, A: 255}, 1),
		buildProceduralClip("ocean-glow", "Ocean Glow", "Tidal caustics · 2.4s loop",
			color.NRGBA{R: 10, G: 24, B: 34, A: 255},
			color.NRGBA{R: 24, G: 168, B: 186, A: 255},
			color.NRGBA{R: 148, G: 248, B: 212, A: 255}, 2),
	}

	return &mediaState{
		artworks:      artworks,
		clips:         clips,
		audioPos:      42 * time.Second,
		audioBuffered: 0.72,
		showQueue:     true,
		lightboxItems: lbItems,
	}
}

func (m *mediaState) close() {
	for i := range m.clips {
		if m.clips[i].player != nil {
			m.clips[i].player.Close()
			m.clips[i].player = nil
		}
	}
}

func (m *mediaState) activeArtwork() *mediaArtwork {
	idx := m.activeTrack
	if idx < 0 || idx >= len(m.artworks) {
		idx = 0
	}
	return &m.artworks[idx]
}

func (m *mediaState) inspectedArtwork() *mediaArtwork {
	idx := m.selectedArtIdx
	if idx < 0 || idx >= len(m.artworks) {
		idx = m.activeTrack
	}
	if idx < 0 || idx >= len(m.artworks) {
		idx = 0
	}
	return &m.artworks[idx]
}

func (m *mediaState) activeCoverSeed() ui.Color {
	if m.customPaletteSeed.A > 0 {
		return m.customPaletteSeed
	}
	return m.inspectedArtwork().seed
}

func (m *mediaState) currentTrackInfo() media.TrackInfo {
	art := m.activeArtwork()
	return media.TrackInfo{
		Title:        art.title,
		Artist:       art.artist,
		Album:        art.album,
		Artwork:      art.bitmap,
		ArtworkImage: art.raw,
		Position:     m.audioPos,
		Duration:     art.duration,
		Buffered:     m.audioBuffered,
	}
}

func (m *mediaState) activeVideoPlayer() *ui.VideoPlayer {
	if len(m.clips) == 0 {
		return nil
	}
	idx := m.activeClip
	if idx < 0 || idx >= len(m.clips) {
		idx = 0
	}
	return m.clips[idx].player
}

func (m *mediaState) selectClip(idx int) {
	if idx < 0 || idx >= len(m.clips) || idx == m.activeClip {
		return
	}
	wasPlaying := false
	if cur := m.activeVideoPlayer(); cur != nil {
		wasPlaying = cur.Playing()
		cur.Pause()
	}
	m.activeClip = idx
	if next := m.activeVideoPlayer(); next != nil {
		next.Seek(0)
		if wasPlaying {
			next.Play()
		} else {
			next.Pause()
		}
	}
}

func (m *mediaState) pauseVideo() {
	if p := m.activeVideoPlayer(); p != nil && p.Playing() {
		p.Pause()
	}
}

func (m *mediaState) selectTrack(idx int) {
	if idx < 0 || idx >= len(m.artworks) {
		return
	}
	m.activeTrack = idx
	m.selectedArtIdx = idx
	m.customPaletteSeed = ui.Color{}
	m.audioPos = 0
}

func (m *mediaState) nextTrack() {
	if len(m.artworks) == 0 {
		return
	}
	if m.shuffle && len(m.artworks) > 1 {
		next := (m.activeTrack + 2) % len(m.artworks)
		m.selectTrack(next)
		return
	}
	m.selectTrack((m.activeTrack + 1) % len(m.artworks))
}

func (m *mediaState) prevTrack() {
	if len(m.artworks) == 0 {
		return
	}
	if m.audioPos > 5*time.Second {
		m.audioPos = 0
		return
	}
	next := m.activeTrack - 1
	if next < 0 {
		next = len(m.artworks) - 1
	}
	m.selectTrack(next)
}

func (m *mediaState) moveTrackInQueue(idx, delta int) {
	target := idx + delta
	if idx < 0 || idx >= len(m.artworks) || target < 0 || target >= len(m.artworks) {
		return
	}
	m.artworks[idx], m.artworks[target] = m.artworks[target], m.artworks[idx]
	m.lightboxItems[idx], m.lightboxItems[target] = m.lightboxItems[target], m.lightboxItems[idx]
	switch m.activeTrack {
	case idx:
		m.activeTrack = target
	case target:
		m.activeTrack = idx
	}
	switch m.selectedArtIdx {
	case idx:
		m.selectedArtIdx = target
	case target:
		m.selectedArtIdx = idx
	}
}

func (m *mediaState) filteredTrackIndices() []int {
	q := strings.ToLower(strings.TrimSpace(m.queueFilter))
	out := make([]int, 0, len(m.artworks))
	for i := range m.artworks {
		art := &m.artworks[i]
		if q != "" &&
			!strings.Contains(strings.ToLower(art.title), q) &&
			!strings.Contains(strings.ToLower(art.artist), q) &&
			!strings.Contains(strings.ToLower(art.album), q) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func (m *mediaState) filteredGalleryIndices() []int {
	out := make([]int, 0, len(m.artworks))
	for i := range m.artworks {
		switch m.galleryToneFilter {
		case toneFilterWarm:
			if !m.artworks[i].warm {
				continue
			}
		case toneFilterCool:
			if m.artworks[i].warm {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}

func (m *mediaState) openLightbox(idx int) {
	if idx < 0 || idx >= len(m.artworks) {
		return
	}
	m.selectedArtIdx = idx
	m.customPaletteSeed = ui.Color{}
	m.lightboxIdx = idx
	m.lightboxOpenIdx = idx
	m.lightboxOpen = true
}

func (m *mediaState) activeLightboxMatchKey() any {
	if !m.lightboxOpen || m.lightboxOpenIdx < 0 || m.lightboxOpenIdx >= len(m.artworks) {
		return nil
	}
	return m.artworks[m.lightboxOpenIdx].matchKey
}

func (m *mediaState) advanceIfDue(now time.Time) {
	if !m.audioPlaying {
		m.lastAudioTick = time.Time{}
		return
	}
	if m.lastAudioTick.IsZero() {
		m.lastAudioTick = now
		return
	}
	elapsed := now.Sub(m.lastAudioTick)
	if elapsed < audioTickInterval {
		return
	}
	m.lastAudioTick = now
	m.audioPos += elapsed
	dur := m.activeArtwork().duration
	if dur > 0 && m.audioPos >= dur {
		if m.repeat {
			m.audioPos = 0
		} else {
			m.nextTrack()
		}
	}
}

func (m *mediaState) scheduleNext(c *ui.Context, now time.Time) {
	if !m.audioPlaying {
		m.lastAudioTick = time.Time{}
		return
	}
	if m.lastAudioTick.IsZero() {
		m.lastAudioTick = now
	}
	rem := audioTickInterval - now.Sub(m.lastAudioTick)
	if rem <= 0 {
		rem = audioTickInterval
	}
	c.After(rem)
}

func synthesizeArtwork(w, h int, r artRecipe) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fw, fh := float64(w), float64(h)
	for y := range h {
		ny := float64(y) / max(1, fh-1)
		base := lerpNRGBA(r.bgTop, r.bgBottom, math.Pow(ny, 0.85))
		for x := range w {
			nx := float64(x) / max(1, fw-1)
			col := base

			switch r.style {
			case 0:
				// Radiant horizon sun + concentric sound rings.
				dx, dy := nx-0.54, ny-0.44
				dist := math.Hypot(dx, dy)
				orb := clamp01(1 - dist/0.46)
				col = lerpNRGBA(col, r.accentB, 0.45*math.Sin(ny*math.Pi*1.4)*math.Sin(nx*math.Pi))
				col = lerpNRGBA(col, r.accentA, math.Pow(orb, 1.5)*0.9)
				ring := math.Exp(-math.Pow((dist-0.28)/0.045, 2))
				col = lerpNRGBA(col, r.glow, ring*0.65)
				if dist < 0.16 {
					core := clamp01(1 - dist/0.16)
					col = lerpNRGBA(col, r.glow, core*0.85)
				}
			case 1:
				// Layered organic sine waves + luminous caustic core.
				wave1 := 0.52 + 0.14*math.Sin(nx*math.Pi*2.4+0.6)
				wave2 := 0.68 + 0.11*math.Cos(nx*math.Pi*3.1)
				if ny > wave1 {
					d := clamp01((ny - wave1) * 3.5)
					col = lerpNRGBA(col, r.accentB, 0.55*(1-0.4*d))
				}
				if ny > wave2 {
					d := clamp01((ny - wave2) * 4.0)
					col = lerpNRGBA(col, r.accentA, 0.48*(1-0.5*d))
				}
				dist := math.Hypot(nx-0.36, ny-0.34)
				halo := math.Exp(-math.Pow(dist/0.24, 2))
				col = lerpNRGBA(col, r.glow, halo*0.75)
			default:
				// Diagonal prism beam + petal/facet glow.
				band := math.Exp(-math.Pow(((nx*0.82+ny*0.58)-0.72)/0.16, 2))
				col = lerpNRGBA(col, r.accentB, band*0.75)
				dist := math.Hypot(nx-0.5, ny-0.5)
				angle := math.Atan2(ny-0.5, nx-0.5)
				petal := 0.5 + 0.5*math.Cos(angle*5)
				bloom := clamp01(1 - dist/(0.34+0.12*petal))
				col = lerpNRGBA(col, r.accentA, bloom*0.82)
				core := math.Exp(-math.Pow(dist/0.14, 2))
				col = lerpNRGBA(col, r.glow, core*0.85)
			}

			img.SetRGBA(x, y, color.RGBA{R: col.R, G: col.G, B: col.B, A: 255})
		}
	}
	return img
}

func buildProceduralClip(id, title, subtitle string, bg, accentA, accentB color.NRGBA, mode int) mediaClip {
	const (
		w      = 192
		h      = 108
		frames = 24
	)
	pal := color.Palette{
		color.RGBA{R: bg.R, G: bg.G, B: bg.B, A: 255},
		color.RGBA{R: accentA.R, G: accentA.G, B: accentA.B, A: 255},
		color.RGBA{R: accentB.R, G: accentB.G, B: accentB.B, A: 255},
	}
	for s := 1; s <= 5; s++ {
		t := float64(s) / 6.0
		m := lerpNRGBA(bg, accentA, t)
		pal = append(pal, color.RGBA{R: m.R, G: m.G, B: m.B, A: 255})
	}
	for s := 1; s <= 4; s++ {
		t := float64(s) / 5.0
		m := lerpNRGBA(accentA, accentB, t)
		pal = append(pal, color.RGBA{R: m.R, G: m.G, B: m.B, A: 255})
	}

	g := &gif.GIF{}
	for f := range frames {
		phase := float64(f) / float64(frames) * 2 * math.Pi
		frame := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for y := range h {
			ny := float64(y) / float64(h-1)
			for x := range w {
				nx := float64(x) / float64(w-1)
				var val float64
				switch mode {
				case 0:
					// Animated spectrum wave bars + drifting solar disc.
					bar := int(nx * 16)
					barH := 0.25 + 0.55*(0.5+0.5*math.Sin(phase+float64(bar)*0.55))
					inBar := (nx*16 - float64(bar)) < 0.72
					if inBar && (1-ny) < barH {
						val = 0.45 + 0.55*(1-ny)
					} else {
						orbX := 0.5 + 0.28*math.Cos(phase)
						orbY := 0.34 + 0.12*math.Sin(phase*2)
						d := math.Hypot((nx-orbX)*1.6, ny-orbY)
						val = clamp01(1-d/0.45) * 0.65
					}
				case 1:
					// Rotating orbital rings + pulsing core.
					cx, cy := (nx-0.5)*1.77, ny-0.5
					d := math.Hypot(cx, cy)
					ang := math.Atan2(cy, cx)
					ring1 := math.Exp(-math.Pow((d-0.32)/0.045, 2))
					ring2 := math.Exp(-math.Pow((d-0.18)/0.035, 2))
					satX := 0.32 * math.Cos(phase)
					satY := 0.32 * math.Sin(phase)
					sat := clamp01(1 - math.Hypot(cx-satX, cy-satY)/0.11)
					sweep := 0.5 + 0.5*math.Cos(ang*3-phase*2)
					val = clamp01(ring1*0.6 + ring2*(0.4+0.4*sweep) + sat*0.95)
				default:
					// Interference wave caustics.
					w1 := math.Sin(nx*10 + phase)
					w2 := math.Cos(ny*7 - phase)
					w3 := math.Sin((nx+ny)*8 + phase*2)
					val = clamp01((w1 + w2 + w3 + 3) / 6.0)
				}
				idx := uint8(clampInt(int(math.Round(val*float64(len(pal)-1))), 0, len(pal)-1))
				frame.SetColorIndex(x, y, idx)
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return mediaClip{id: id, title: title, subtitle: subtitle, err: err}
	}
	dec, err := ui.NewGIFDecoder(buf.Bytes())
	if err != nil {
		return mediaClip{id: id, title: title, subtitle: subtitle, err: err}
	}
	vp, err := ui.NewVideoPlayer(dec)
	if err != nil {
		return mediaClip{id: id, title: title, subtitle: subtitle, err: err}
	}
	vp.SetLoop(true)
	vp.Pause()
	return mediaClip{
		id:       id,
		title:    title,
		subtitle: subtitle,
		player:   vp,
	}
}

func lerpNRGBA(a, b color.NRGBA, t float64) color.NRGBA {
	t = clamp01(t)
	return color.NRGBA{
		R: uint8(math.Round(float64(a.R)*(1-t) + float64(b.R)*t)),
		G: uint8(math.Round(float64(a.G)*(1-t) + float64(b.G)*t)),
		B: uint8(math.Round(float64(a.B)*(1-t) + float64(b.B)*t)),
		A: 255,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mediaColorHex(c ui.Color) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func formatMediaDuration(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}
	sec := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}
