package tokens

import (
	"cmp"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// defaultExtractSeed is Material's baseline purple (#6750A4), returned when
// an image is nil, empty, transparent, or filtered to zero chromatic candidates.
var defaultExtractSeed = ui.RGBA(0x67, 0x50, 0xA4, 1)

const (
	extractMaxSamples     = 128 * 128
	extractDefaultColors  = 128
	extractMaxColorsLimit = 256
	extractDefaultDesired = 4
	extractMaxDesired     = 16
	wuBits                = 5
	wuIndexCount          = (1 << wuBits) + 1 // 33
	wsmeansMaxIterations  = 10
	scoreTargetChroma     = 48.0
	scoreWeightProportion = 0.7
	scoreWeightChromaHigh = 0.3
	scoreWeightChromaLow  = 0.1
	scoreCutoffChroma     = 5.0
	scoreCutoffProportion = 0.01
)

// ExtractOptions configures image seed and palette extraction via QuantizerCelebi
// (Wu + CIE L*a*b* WSMeans) and Material Score ranking.
type ExtractOptions struct {
	// MaxColors bounds the number of quantized palette clusters (default 128,
	// clamped to 1..256).
	MaxColors int
	// Desired bounds how many diverse top-scored colors PaletteFromImage returns
	// (default 4, clamped to 1..16).
	Desired int
	// Fallback is returned when img is nil, empty, fully transparent, or
	// contains no chromatic colors under filtering. Defaults to #6750A4.
	Fallback ui.Color
	// AllowLowChroma disables Material Score's chroma (< 5) and low-proportion
	// (<= 1%) filtering so grayscale/low-chroma covers can return a dominant tone.
	AllowLowChroma bool
}

func normalizeExtractOptions(opts ...ExtractOptions) ExtractOptions {
	if len(opts) > 1 {
		panic("tokens: image color extraction takes at most one ExtractOptions")
	}
	var o ExtractOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.MaxColors <= 0 {
		o.MaxColors = extractDefaultColors
	} else if o.MaxColors > extractMaxColorsLimit {
		o.MaxColors = extractMaxColorsLimit
	}
	if o.Desired <= 0 {
		o.Desired = extractDefaultDesired
	} else if o.Desired > extractMaxDesired {
		o.Desired = extractMaxDesired
	}
	if o.Fallback.A == 0 {
		o.Fallback = defaultExtractSeed
	}
	return o
}

// SeedFromImage extracts the single highest-scoring Material HCT seed color
// from img using pure-Go QuantizerCelebi (Wu + CIE L*a*b* WSMeans) and
// Material Score ranking. Returns Fallback (#6750A4 by default) when img is
// nil, empty, fully transparent, or lacks chromatic candidates.
func SeedFromImage(img image.Image, opts ...ExtractOptions) ui.Color {
	pal := PaletteFromImage(img, opts...)
	return pal[0]
}

// PaletteFromImage extracts up to Desired diverse, Material-scored seed colors
// ordered from highest score to lowest. Always returns at least one color
// (Fallback when no valid candidates survive).
// The sampling grid visits at most 16,384 pixels, including for elongated images.
func PaletteFromImage(img image.Image, opts ...ExtractOptions) []ui.Color {
	o := normalizeExtractOptions(opts...)
	pixels := sampleImagePixels(img)
	if len(pixels) == 0 {
		return []ui.Color{o.Fallback}
	}
	quantized := quantizeCelebi(pixels, o.MaxColors)
	if len(quantized) == 0 {
		return []ui.Color{o.Fallback}
	}
	return scoreQuantizedColors(quantized, o)
}

type rgbCount struct {
	rgb   uint32
	count int
}

func packRGB(r, g, b uint8) uint32 {
	return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

func unpackRGB(rgb uint32) (r, g, b uint8) {
	return uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb)
}

func colorFromRGB24(rgb uint32) ui.Color {
	r, g, b := unpackRGB(rgb)
	return ui.RGB(r, g, b)
}

func sampleImagePixels(img image.Image) []rgbCount {
	if img == nil {
		return nil
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	stepX, stepY := 1, 1
	if area := float64(w) * float64(h); area > extractMaxSamples {
		stepX = int(math.Ceil(math.Sqrt(area / extractMaxSamples)))
		stepY = stepX
	}
	cols, rows := 1+(w-1)/stepX, 1+(h-1)/stepY
	if cols > extractMaxSamples/rows {
		// Rounding each axis to a whole sample can put the grid over budget.
		// Keep the short axis intact and increase the stride on the long axis.
		if w >= h {
			stepX = 1 + (w-1)/(extractMaxSamples/rows)
			cols = 1 + (w-1)/stepX
		} else {
			stepY = 1 + (h-1)/(extractMaxSamples/cols)
			rows = 1 + (h-1)/stepY
		}
	}

	counts := make(map[uint32]int, min(cols*rows, 4096))
	addOpaque := func(r, g, b uint8) {
		counts[packRGB(r, g, b)]++
	}

	switch m := img.(type) {
	case *image.RGBA:
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			rowOff := m.PixOffset(bounds.Min.X, y)
			for col := range cols {
				x := bounds.Min.X + col*stepX
				off := rowOff + (x-bounds.Min.X)*4
				a := m.Pix[off+3]
				if a < 128 {
					continue
				}
				r, g, b := m.Pix[off], m.Pix[off+1], m.Pix[off+2]
				if a < 255 {
					ai := int(a)
					r = uint8(min(255, int(r)*255/ai))
					g = uint8(min(255, int(g)*255/ai))
					b = uint8(min(255, int(b)*255/ai))
				}
				addOpaque(r, g, b)
			}
		}
	case *image.NRGBA:
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			rowOff := m.PixOffset(bounds.Min.X, y)
			for col := range cols {
				x := bounds.Min.X + col*stepX
				off := rowOff + (x-bounds.Min.X)*4
				if m.Pix[off+3] < 128 {
					continue
				}
				addOpaque(m.Pix[off], m.Pix[off+1], m.Pix[off+2])
			}
		}
	case *image.YCbCr:
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			for col := range cols {
				x := bounds.Min.X + col*stepX
				yi := m.YOffset(x, y)
				ci := m.COffset(x, y)
				r, g, b := color.YCbCrToRGB(m.Y[yi], m.Cb[ci], m.Cr[ci])
				addOpaque(r, g, b)
			}
		}
	case *image.Paletted:
		palRGB := make([]uint32, len(m.Palette))
		palValid := make([]bool, len(m.Palette))
		for i, pc := range m.Palette {
			nr := color.NRGBAModel.Convert(pc).(color.NRGBA)
			if nr.A >= 128 {
				palRGB[i] = packRGB(nr.R, nr.G, nr.B)
				palValid[i] = true
			}
		}
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			rowOff := m.PixOffset(bounds.Min.X, y)
			for col := range cols {
				x := bounds.Min.X + col*stepX
				idx := int(m.Pix[rowOff+(x-bounds.Min.X)])
				if idx < len(palValid) && palValid[idx] {
					counts[palRGB[idx]]++
				}
			}
		}
	case *image.Gray:
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			rowOff := m.PixOffset(bounds.Min.X, y)
			for col := range cols {
				x := bounds.Min.X + col*stepX
				v := m.Pix[rowOff+(x-bounds.Min.X)]
				addOpaque(v, v, v)
			}
		}
	default:
		for row := range rows {
			y := bounds.Min.Y + row*stepY
			for col := range cols {
				x := bounds.Min.X + col*stepX
				r16, g16, b16, a16 := img.At(x, y).RGBA()
				if a16 < 0x8000 {
					continue
				}
				if a16 == 0xffff {
					addOpaque(uint8(r16>>8), uint8(g16>>8), uint8(b16>>8))
					continue
				}
				r := uint8(min(255, int(r16*255/a16)))
				g := uint8(min(255, int(g16*255/a16)))
				b := uint8(min(255, int(b16*255/a16)))
				addOpaque(r, g, b)
			}
		}
	}

	if len(counts) == 0 {
		return nil
	}
	out := make([]rgbCount, 0, len(counts))
	for rgb, c := range counts {
		out = append(out, rgbCount{rgb: rgb, count: c})
	}
	slices.SortFunc(out, func(a, b rgbCount) int {
		return cmp.Compare(a.rgb, b.rgb)
	})
	return out
}

func quantizeCelebi(pixels []rgbCount, maxColors int) []rgbCount {
	wuClusters := quantizeWu(pixels, maxColors)
	if len(wuClusters) == 0 {
		return nil
	}
	return quantizeWSMeans(pixels, wuClusters, maxColors)
}

type wuAxis int

const (
	wuRed wuAxis = iota
	wuGreen
	wuBlue
)

type wuBox struct {
	r0, r1 int
	g0, g1 int
	b0, b1 int
	vol    int
}

type wuQuantizer struct {
	weights  [wuIndexCount][wuIndexCount][wuIndexCount]int32
	momentsR [wuIndexCount][wuIndexCount][wuIndexCount]int32
	momentsG [wuIndexCount][wuIndexCount][wuIndexCount]int32
	momentsB [wuIndexCount][wuIndexCount][wuIndexCount]int32
	moments  [wuIndexCount][wuIndexCount][wuIndexCount]int64
}

func wuIndex(r, g, b uint8) (int, int, int) {
	shift := 8 - wuBits
	return int(r>>shift) + 1, int(g>>shift) + 1, int(b>>shift) + 1
}

func quantizeWu(pixels []rgbCount, maxColors int) []uint32 {
	q := &wuQuantizer{}
	for _, px := range pixels {
		r, g, b := unpackRGB(px.rgb)
		ir, ig, ib := wuIndex(r, g, b)
		c := int32(px.count)
		ri, gi, bi := int32(r), int32(g), int32(b)
		q.weights[ir][ig][ib] += c
		q.momentsR[ir][ig][ib] += ri * c
		q.momentsG[ir][ig][ib] += gi * c
		q.momentsB[ir][ig][ib] += bi * c
		q.moments[ir][ig][ib] += int64(px.count) * int64(ri*ri+gi*gi+bi*bi)
	}

	// Build 3D prefix sum tables.
	for r := 1; r < wuIndexCount; r++ {
		var area [wuIndexCount]int32
		var areaR, areaG, areaB [wuIndexCount]int32
		var area2 [wuIndexCount]int64
		for g := 1; g < wuIndexCount; g++ {
			var line, lineR, lineG, lineB int32
			var line2 int64
			for b := 1; b < wuIndexCount; b++ {
				line += q.weights[r][g][b]
				lineR += q.momentsR[r][g][b]
				lineG += q.momentsG[r][g][b]
				lineB += q.momentsB[r][g][b]
				line2 += q.moments[r][g][b]

				area[b] += line
				areaR[b] += lineR
				areaG[b] += lineG
				areaB[b] += lineB
				area2[b] += line2

				q.weights[r][g][b] = q.weights[r-1][g][b] + area[b]
				q.momentsR[r][g][b] = q.momentsR[r-1][g][b] + areaR[b]
				q.momentsG[r][g][b] = q.momentsG[r-1][g][b] + areaG[b]
				q.momentsB[r][g][b] = q.momentsB[r-1][g][b] + areaB[b]
				q.moments[r][g][b] = q.moments[r-1][g][b] + area2[b]
			}
		}
	}

	cubes := make([]wuBox, maxColors)
	cubes[0] = wuBox{r1: wuIndexCount - 1, g1: wuIndexCount - 1, b1: wuIndexCount - 1}
	vv := make([]float64, maxColors)
	next := 0
	generated := 1
	for i := 1; i < maxColors; i++ {
		if q.cut(&cubes[next], &cubes[i]) {
			if cubes[next].vol > 1 {
				vv[next] = q.variance(cubes[next])
			} else {
				vv[next] = 0
			}
			if cubes[i].vol > 1 {
				vv[i] = q.variance(cubes[i])
			} else {
				vv[i] = 0
			}
			generated = i + 1
		} else {
			vv[next] = 0
			i--
		}
		next = 0
		temp := vv[0]
		for k := 1; k < generated; k++ {
			if vv[k] > temp {
				temp = vv[k]
				next = k
			}
		}
		if temp <= 0 {
			break
		}
	}

	out := make([]uint32, 0, generated)
	for k := 0; k < generated; k++ {
		w := volumeI32(&q.weights, cubes[k])
		if w <= 0 {
			continue
		}
		r := uint8(clampInt(int(volumeI32(&q.momentsR, cubes[k])/w), 0, 255))
		g := uint8(clampInt(int(volumeI32(&q.momentsG, cubes[k])/w), 0, 255))
		b := uint8(clampInt(int(volumeI32(&q.momentsB, cubes[k])/w), 0, 255))
		out = append(out, packRGB(r, g, b))
	}
	return out
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

func volumeI32(arr *[wuIndexCount][wuIndexCount][wuIndexCount]int32, c wuBox) int32 {
	return arr[c.r1][c.g1][c.b1] -
		arr[c.r1][c.g1][c.b0] -
		arr[c.r1][c.g0][c.b1] +
		arr[c.r1][c.g0][c.b0] -
		arr[c.r0][c.g1][c.b1] +
		arr[c.r0][c.g1][c.b0] +
		arr[c.r0][c.g0][c.b1] -
		arr[c.r0][c.g0][c.b0]
}

func volumeI64(arr *[wuIndexCount][wuIndexCount][wuIndexCount]int64, c wuBox) int64 {
	return arr[c.r1][c.g1][c.b1] -
		arr[c.r1][c.g1][c.b0] -
		arr[c.r1][c.g0][c.b1] +
		arr[c.r1][c.g0][c.b0] -
		arr[c.r0][c.g1][c.b1] +
		arr[c.r0][c.g1][c.b0] +
		arr[c.r0][c.g0][c.b1] -
		arr[c.r0][c.g0][c.b0]
}

func bottomI32(arr *[wuIndexCount][wuIndexCount][wuIndexCount]int32, c wuBox, dir wuAxis) int32 {
	switch dir {
	case wuRed:
		return -arr[c.r0][c.g1][c.b1] + arr[c.r0][c.g1][c.b0] + arr[c.r0][c.g0][c.b1] - arr[c.r0][c.g0][c.b0]
	case wuGreen:
		return -arr[c.r1][c.g0][c.b1] + arr[c.r1][c.g0][c.b0] + arr[c.r0][c.g0][c.b1] - arr[c.r0][c.g0][c.b0]
	default:
		return -arr[c.r1][c.g1][c.b0] + arr[c.r1][c.g0][c.b0] + arr[c.r0][c.g1][c.b0] - arr[c.r0][c.g0][c.b0]
	}
}

func topI32(arr *[wuIndexCount][wuIndexCount][wuIndexCount]int32, c wuBox, dir wuAxis, pos int) int32 {
	switch dir {
	case wuRed:
		return arr[pos][c.g1][c.b1] - arr[pos][c.g1][c.b0] - arr[pos][c.g0][c.b1] + arr[pos][c.g0][c.b0]
	case wuGreen:
		return arr[c.r1][pos][c.b1] - arr[c.r1][pos][c.b0] - arr[c.r0][pos][c.b1] + arr[c.r0][pos][c.b0]
	default:
		return arr[c.r1][c.g1][pos] - arr[c.r1][c.g0][pos] - arr[c.r0][c.g1][pos] + arr[c.r0][c.g0][pos]
	}
}

func (q *wuQuantizer) variance(c wuBox) float64 {
	w := float64(volumeI32(&q.weights, c))
	if w <= 0 {
		return 0
	}
	dr := float64(volumeI32(&q.momentsR, c))
	dg := float64(volumeI32(&q.momentsG, c))
	db := float64(volumeI32(&q.momentsB, c))
	m2 := float64(volumeI64(&q.moments, c))
	return m2 - (dr*dr+dg*dg+db*db)/w
}

func (q *wuQuantizer) maximize(c wuBox, dir wuAxis, first, last int, wholeR, wholeG, wholeB, wholeW int32) (cut int, maxVal float64) {
	baseR := bottomI32(&q.momentsR, c, dir)
	baseG := bottomI32(&q.momentsG, c, dir)
	baseB := bottomI32(&q.momentsB, c, dir)
	baseW := bottomI32(&q.weights, c, dir)
	cut = -1
	for i := first; i < last; i++ {
		halfW := baseW + topI32(&q.weights, c, dir, i)
		if halfW == 0 {
			continue
		}
		remW := wholeW - halfW
		if remW == 0 {
			break
		}
		halfR := float64(baseR + topI32(&q.momentsR, c, dir, i))
		halfG := float64(baseG + topI32(&q.momentsG, c, dir, i))
		halfB := float64(baseB + topI32(&q.momentsB, c, dir, i))
		temp := (halfR*halfR + halfG*halfG + halfB*halfB) / float64(halfW)

		remR := float64(wholeR) - halfR
		remG := float64(wholeG) - halfG
		remB := float64(wholeB) - halfB
		temp += (remR*remR + remG*remG + remB*remB) / float64(remW)
		if temp > maxVal {
			maxVal = temp
			cut = i
		}
	}
	return cut, maxVal
}

func (q *wuQuantizer) cut(one, two *wuBox) bool {
	wholeR := volumeI32(&q.momentsR, *one)
	wholeG := volumeI32(&q.momentsG, *one)
	wholeB := volumeI32(&q.momentsB, *one)
	wholeW := volumeI32(&q.weights, *one)

	cutR, maxR := q.maximize(*one, wuRed, one.r0+1, one.r1, wholeR, wholeG, wholeB, wholeW)
	cutG, maxG := q.maximize(*one, wuGreen, one.g0+1, one.g1, wholeR, wholeG, wholeB, wholeW)
	cutB, maxB := q.maximize(*one, wuBlue, one.b0+1, one.b1, wholeR, wholeG, wholeB, wholeW)

	var dir wuAxis
	if maxR >= maxG && maxR >= maxB {
		dir = wuRed
		if cutR < 0 {
			return false
		}
	} else if maxG >= maxR && maxG >= maxB {
		dir = wuGreen
	} else {
		dir = wuBlue
	}

	two.r1, two.g1, two.b1 = one.r1, one.g1, one.b1
	switch dir {
	case wuRed:
		two.r0, one.r1 = cutR, cutR
		two.g0, two.b0 = one.g0, one.b0
	case wuGreen:
		two.g0, one.g1 = cutG, cutG
		two.r0, two.b0 = one.r0, one.b0
	default:
		two.b0, one.b1 = cutB, cutB
		two.r0, two.g0 = one.r0, one.g0
	}
	one.vol = (one.r1 - one.r0) * (one.g1 - one.g0) * (one.b1 - one.b0)
	two.vol = (two.r1 - two.r0) * (two.g1 - two.g0) * (two.b1 - two.b0)
	return true
}

type labPoint struct {
	l, a, b float64
}

func labFromRGB24(rgb uint32) labPoint {
	l, a, b := LabFromXYZ(XYZFromColor(colorFromRGB24(rgb)))
	return labPoint{l: l, a: a, b: b}
}

func labDist2(p, q labPoint) float64 {
	dl := p.l - q.l
	da := p.a - q.a
	db := p.b - q.b
	return dl*dl + da*da + db*db
}

func quantizeWSMeans(pixels []rgbCount, starting []uint32, maxColors int) []rgbCount {
	// Deduplicate starting centroids preserving deterministic order.
	seenStart := make(map[uint32]struct{}, len(starting))
	clusterRGB := make([]uint32, 0, min(len(starting), maxColors))
	for _, rgb := range starting {
		if _, ok := seenStart[rgb]; ok {
			continue
		}
		seenStart[rgb] = struct{}{}
		clusterRGB = append(clusterRGB, rgb)
		if len(clusterRGB) >= maxColors {
			break
		}
	}
	k := len(clusterRGB)
	if k == 0 {
		return nil
	}

	points := make([]labPoint, len(pixels))
	clusterIndices := make([]int, len(pixels))
	for i, px := range pixels {
		points[i] = labFromRGB24(px.rgb)
	}

	clusters := make([]labPoint, k)
	for i, rgb := range clusterRGB {
		clusters[i] = labFromRGB24(rgb)
	}

	// Assign initial clusters deterministically.
	for i, pt := range points {
		bestIdx := 0
		bestDist := labDist2(pt, clusters[0])
		for c := 1; c < k; c++ {
			if d := labDist2(pt, clusters[c]); d < bestDist {
				bestDist = d
				bestIdx = c
			}
		}
		clusterIndices[i] = bestIdx
	}

	distMat := make([]float64, k*k)
	type idxDist struct {
		idx  int
		dist float64
	}
	sortedMat := make([]idxDist, k*k)
	pixelCounts := make([]int, k)
	sumL := make([]float64, k)
	sumA := make([]float64, k)
	sumB := make([]float64, k)

	for iter := 0; iter < wsmeansMaxIterations; iter++ {
		for i := 0; i < k; i++ {
			for j := i + 1; j < k; j++ {
				d := labDist2(clusters[i], clusters[j])
				distMat[i*k+j] = d
				distMat[j*k+i] = d
			}
		}
		for i := 0; i < k; i++ {
			row := sortedMat[i*k : (i+1)*k]
			for j := 0; j < k; j++ {
				row[j] = idxDist{idx: j, dist: distMat[i*k+j]}
			}
			slices.SortFunc(row, func(a, b idxDist) int {
				if c := cmp.Compare(a.dist, b.dist); c != 0 {
					return c
				}
				return cmp.Compare(a.idx, b.idx)
			})
		}

		changes := 0
		for i, pt := range points {
			prevIdx := clusterIndices[i]
			prevDist := labDist2(pt, clusters[prevIdx])
			bestIdx := -1
			bestDist := prevDist
			row := sortedMat[prevIdx*k : (prevIdx+1)*k]
			for j := 0; j < k; j++ {
				if row[j].dist >= 4*prevDist {
					break
				}
				cand := row[j].idx
				if d := labDist2(pt, clusters[cand]); d < bestDist {
					bestDist = d
					bestIdx = cand
				}
			}
			if bestIdx != -1 && math.Abs(bestDist-prevDist) > 3.0 {
				changes++
				clusterIndices[i] = bestIdx
			}
		}
		if changes == 0 && iter != 0 {
			break
		}

		clear(pixelCounts)
		clear(sumL)
		clear(sumA)
		clear(sumB)
		for i, px := range pixels {
			c := clusterIndices[i]
			w := float64(px.count)
			pixelCounts[c] += px.count
			sumL[c] += points[i].l * w
			sumA[c] += points[i].a * w
			sumB[c] += points[i].b * w
		}
		for c := 0; c < k; c++ {
			if pixelCounts[c] == 0 {
				clusters[c] = labPoint{}
				continue
			}
			w := float64(pixelCounts[c])
			clusters[c] = labPoint{l: sumL[c] / w, a: sumA[c] / w, b: sumB[c] / w}
		}
	}

	merged := make(map[uint32]int, k)
	for c := 0; c < k; c++ {
		cnt := pixelCounts[c]
		if cnt <= 0 {
			continue
		}
		col := ColorFromXYZ(XYZFromLab(clusters[c].l, clusters[c].a, clusters[c].b))
		merged[packRGB(col.R, col.G, col.B)] += cnt
	}
	if len(merged) == 0 {
		return nil
	}
	out := make([]rgbCount, 0, len(merged))
	for rgb, cnt := range merged {
		out = append(out, rgbCount{rgb: rgb, count: cnt})
	}
	slices.SortFunc(out, func(a, b rgbCount) int {
		return cmp.Compare(a.rgb, b.rgb)
	})
	return out
}

func hueDifferenceDegrees(a, b float64) float64 {
	return 180.0 - math.Abs(math.Abs(a-b)-180.0)
}

func scoreQuantizedColors(quantized []rgbCount, o ExtractOptions) []ui.Color {
	type hctCandidate struct {
		rgb   uint32
		color ui.Color
		hct   HCT
		score float64
	}

	candidates := make([]hctCandidate, 0, len(quantized))
	var huePop [360]float64
	totalPop := 0.0
	for _, q := range quantized {
		col := colorFromRGB24(q.rgb)
		hct := HCTFromColor(col)
		candidates = append(candidates, hctCandidate{rgb: q.rgb, color: col, hct: hct})
		hue := int(math.Floor(SanitizeDegrees(hct.Hue))) % 360
		huePop[hue] += float64(q.count)
		totalPop += float64(q.count)
	}
	if totalPop <= 0 {
		return []ui.Color{o.Fallback}
	}

	excited := make([]float64, 360)
	for hue := 0; hue < 360; hue++ {
		prop := huePop[hue] / totalPop
		for d := -14; d < 16; d++ {
			idx := (hue + d + 360) % 360
			excited[idx] += prop
		}
	}

	filtered := make([]hctCandidate, 0, len(candidates))
	for _, cand := range candidates {
		hue := int(math.Round(SanitizeDegrees(cand.hct.Hue))) % 360
		prop := excited[hue]
		if !o.AllowLowChroma && (cand.hct.Chroma < scoreCutoffChroma || prop <= scoreCutoffProportion) {
			continue
		}
		propScore := prop * 100.0 * scoreWeightProportion
		chromaWeight := scoreWeightChromaHigh
		if cand.hct.Chroma < scoreTargetChroma {
			chromaWeight = scoreWeightChromaLow
		}
		chromaScore := (cand.hct.Chroma - scoreTargetChroma) * chromaWeight
		cand.score = propScore + chromaScore
		filtered = append(filtered, cand)
	}
	if len(filtered) == 0 {
		return []ui.Color{o.Fallback}
	}

	slices.SortFunc(filtered, func(a, b hctCandidate) int {
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		return cmp.Compare(a.rgb, b.rgb)
	})

	chosen := make([]hctCandidate, 0, o.Desired)
	for cutoff := 90; cutoff >= 15; cutoff-- {
		chosen = chosen[:0]
		for _, cand := range filtered {
			dupHue := false
			for _, ex := range chosen {
				if hueDifferenceDegrees(cand.hct.Hue, ex.hct.Hue) < float64(cutoff) {
					dupHue = true
					break
				}
			}
			if !dupHue {
				chosen = append(chosen, cand)
				if len(chosen) >= o.Desired {
					break
				}
			}
		}
		if len(chosen) >= o.Desired {
			break
		}
	}
	if len(chosen) == 0 {
		return []ui.Color{o.Fallback}
	}
	out := make([]ui.Color, len(chosen))
	for i, c := range chosen {
		out[i] = c.color
	}
	return out
}
