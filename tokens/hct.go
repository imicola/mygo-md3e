// Package tokens holds the Material Design 3 Expressive design tokens: the
// HCT color space and its CAM16 appearance model, tonal palettes and dynamic
// color schemes, typography, shape and elevation.
//
// The package depends only on the Go standard library and the ui.Color type of
// MyGo's UI toolkit; it knows nothing of layout, state or animation.
package tokens

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The D65 white point, with Y in 0..100.
const (
	whitePointX = 95.047
	whitePointY = 100.0
	whitePointZ = 108.883
)

// CIE L*a*b* constants (CIE 15:2004).
const (
	labEpsilon = 216.0 / 24389.0
	labKappa   = 24389.0 / 27.0
)

// HCT is a color in the HCT color space: the hue and chroma of CAM16 and the
// tone of CIE L*a*b*, which Material uses for lightness.
//
// Hue is in degrees, 0..360; chroma is 0 or more; tone is CIE L* in 0..100,
// where 0 is black and 100 is white.
type HCT struct {
	Hue    float64
	Chroma float64
	Tone   float64
}

// CAM16 is a color as the CAM16 color appearance model sees it under the
// viewing conditions Material's HCT uses: D65 white, a 50% gray background
// and average surround.
type CAM16 struct {
	Hue    float64 // hue angle in degrees, 0..360
	Chroma float64 // colorfulness
	J      float64 // lightness, 0..100
	Q      float64 // brightness
	M      float64 // colorfulness in CAM16-UCS
	S      float64 // saturation
}

// HCTFromColor returns the HCT coordinates of c. Its Tone is the CIE L* of
// c's luminance.
func HCTFromColor(c ui.Color) HCT {
	xyz := XYZFromColor(c)
	cam := cam16FromXYZ(xyz)
	return HCT{Hue: cam.hue, Chroma: cam.chroma, Tone: LstarFromY(xyz[1])}
}

// ToColor returns the sRGB color with the HCT coordinates h. A hue outside
// 0..360 wraps around, and a tone outside 0..100 clamps to black or white.
// A chroma greater than the sRGB gamut holds at h's tone is reduced to the
// largest chroma that fits, as Material's HCT solver does.
func (h HCT) ToColor() ui.Color {
	hue := SanitizeDegrees(h.Hue)
	chroma := math.Max(0, h.Chroma)
	tone := min(max(h.Tone, 0), 100)

	switch {
	case tone <= 0:
		return ui.RGB(0, 0, 0)
	case tone >= 100:
		return ui.RGB(255, 255, 255)
	case chroma < 1e-4:
		return grayColor(tone)
	}

	if lin, ok := solveLinear(hue, chroma, tone); ok {
		return colorFromLinear(lin)
	}

	// The chroma is outside the sRGB gamut at this tone: bisect for the
	// largest chroma that fits.
	lo, hi := 0.0, chroma
	var best [3]float64
	found := false
	for range 24 {
		mid := (lo + hi) / 2
		lin, fits := solveLinear(hue, mid, tone)
		if fits {
			lo, best, found = mid, lin, true
		} else {
			hi = mid
		}
	}
	if !found {
		return grayColor(tone)
	}
	return colorFromLinear(best)
}

// CAM16FromColor returns the CAM16 appearance of c.
func CAM16FromColor(c ui.Color) CAM16 {
	cam := cam16FromXYZ(XYZFromColor(c))
	return CAM16{Hue: cam.hue, Chroma: cam.chroma, J: cam.j, Q: cam.q, M: cam.m, S: cam.s}
}

// ColorFromCAM16 returns the sRGB color with the CAM16 lightness j, chroma
// and hue. A negative lightness or chroma clamps to zero, and a color outside
// the sRGB gamut clamps per channel.
func ColorFromCAM16(j, chroma, hue float64) ui.Color {
	return ColorFromXYZ(xyzFromCAM16(cam16{
		j:      math.Max(0, j),
		chroma: math.Max(0, chroma),
		hue:    hue,
	}))
}

// SanitizeDegrees wraps degrees into 0..360.
func SanitizeDegrees(degrees float64) float64 {
	degrees = math.Mod(degrees, 360)
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

// LinearizeSRGB converts an sRGB channel in 0..1 to linear light in 0..1.
func LinearizeSRGB(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// DelinearizeSRGB converts a linear light channel in 0..1 to sRGB in 0..1.
func DelinearizeSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

// XYZFromColor converts c to CIE XYZ under D65, with Y in 0..100. Alpha is
// ignored.
func XYZFromColor(c ui.Color) [3]float64 {
	r := LinearizeSRGB(float64(c.R) / 255)
	g := LinearizeSRGB(float64(c.G) / 255)
	b := LinearizeSRGB(float64(c.B) / 255)
	return [3]float64{
		(0.41233895*r + 0.35762064*g + 0.18051042*b) * 100,
		(0.2126*r + 0.7152*g + 0.0722*b) * 100,
		(0.01932141*r + 0.11916382*g + 0.95034478*b) * 100,
	}
}

// ColorFromXYZ converts CIE XYZ under D65 (Y in 0..100) to sRGB, clamping
// each channel into the gamut.
func ColorFromXYZ(xyz [3]float64) ui.Color {
	return colorFromLinear(linearRGBFromXYZ(xyz))
}

// LabFromXYZ converts CIE XYZ under D65 (Y in 0..100) to CIE L*a*b*.
func LabFromXYZ(xyz [3]float64) (l, a, b float64) {
	fx := labForward(xyz[0] / whitePointX)
	fy := labForward(xyz[1] / whitePointY)
	fz := labForward(xyz[2] / whitePointZ)
	return 116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)
}

// XYZFromLab converts CIE L*a*b* to CIE XYZ under D65 (Y in 0..100).
func XYZFromLab(l, a, b float64) [3]float64 {
	fy := (l + 16) / 116
	fx := a/500 + fy
	fz := fy - b/200
	return [3]float64{
		whitePointX * labInverse(fx),
		whitePointY * labInverse(fy),
		whitePointZ * labInverse(fz),
	}
}

// YFromLstar converts a CIE L* tone in 0..100 to relative luminance in 0..100.
func YFromLstar(lstar float64) float64 {
	if lstar > 8 {
		return 100 * math.Pow((lstar+16)/116, 3)
	}
	return lstar / labKappa * 100
}

// LstarFromY converts relative luminance in 0..100 to a CIE L* tone.
func LstarFromY(y float64) float64 {
	normalized := y / 100
	if normalized <= labEpsilon {
		return labKappa * normalized
	}
	return 116*math.Cbrt(normalized) - 16
}

// RelativeLuminance returns c's WCAG relative luminance in 0..1. Alpha is
// ignored.
func RelativeLuminance(c ui.Color) float64 {
	return 0.2126*LinearizeSRGB(float64(c.R)/255) +
		0.7152*LinearizeSRGB(float64(c.G)/255) +
		0.0722*LinearizeSRGB(float64(c.B)/255)
}

// ContrastRatio returns the WCAG 2.1 contrast ratio between two colors, from
// 1 (the same color) to 21 (black and white).
func ContrastRatio(a, b ui.Color) float64 {
	la := RelativeLuminance(a)
	lb := RelativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// ContrastRatioOfTones returns the contrast ratio between two tones, as if
// they were gray. Tones outside 0..100 clamp.
func ContrastRatioOfTones(tone1, tone2 float64) float64 {
	tone1 = min(max(tone1, 0), 100)
	tone2 = min(max(tone2, 0), 100)
	la := YFromLstar(tone1)
	lb := YFromLstar(tone2)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05*100) / (lb + 0.05*100)
}

// ContrastTone returns a tone that reaches at least ratio against
// backgroundTone: white (100) or black (0), preferring the one with the
// greater contrast. When neither reaches it, it returns the one with the
// greater contrast.
func ContrastTone(backgroundTone, ratio float64) float64 {
	light := ContrastRatioOfTones(backgroundTone, 100)
	dark := ContrastRatioOfTones(backgroundTone, 0)
	switch {
	case light >= ratio && light >= dark:
		return 100
	case dark >= ratio:
		return 0
	case light >= ratio:
		return 100
	case light >= dark:
		return 100
	default:
		return 0
	}
}

// labForward is the f(t) of CIE L*a*b*.
func labForward(t float64) float64 {
	if t > labEpsilon {
		return math.Cbrt(t)
	}
	return (labKappa*t + 16) / 116
}

// labInverse is the inverse of labForward.
func labInverse(f float64) float64 {
	f3 := f * f * f
	if f3 > labEpsilon {
		return f3
	}
	return (116*f - 16) / labKappa
}

// grayColor returns the gray with the given tone.
func grayColor(tone float64) ui.Color {
	v := channel8(YFromLstar(tone) / 100)
	return ui.RGB(v, v, v)
}

// linearRGBFromXYZ converts CIE XYZ under D65 (Y in 0..100) to linear sRGB in
// 0..1, without clamping.
func linearRGBFromXYZ(xyz [3]float64) [3]float64 {
	x := xyz[0]
	y := xyz[1]
	z := xyz[2]
	// Material's _xyzToSrgb matrix, used as published rather than the exact
	// inverse of the forward matrix, so that rounded channels match it.
	return [3]float64{
		0.032413774792388685*x - 0.015376652402851851*y - 0.0049885366846268053*z,
		-0.009691452513005321*x + 0.018758853451067872*y + 0.0004156585616912061*z,
		0.0005562093689691305*x - 0.0020395524564742123*y + 0.010571799111220335*z,
	}
}

// colorFromLinear converts linear sRGB in 0..1 to an opaque color, clamping
// each channel.
func colorFromLinear(lin [3]float64) ui.Color {
	return ui.RGB(channel8(lin[0]), channel8(lin[1]), channel8(lin[2]))
}

// channel8 converts a linear sRGB channel in 0..1 to an 8-bit sRGB channel,
// clamping it to the gamut.
func channel8(linear float64) uint8 {
	v := DelinearizeSRGB(min(max(linear, 0), 1))
	return uint8(math.Round(min(max(v, 0), 1) * 255))
}

// inGamut reports whether every linear sRGB channel is inside 0..1, allowing
// for floating point noise.
func inGamut(lin [3]float64) bool {
	const epsilon = 1e-4
	for _, v := range lin {
		if v < -epsilon || v > 1+epsilon {
			return false
		}
	}
	return true
}

// solveLinear returns the linear sRGB color with the CAM16 hue and chroma h
// and the tone lstar, and whether such a color exists in the sRGB gamut. The
// CAM16 lightness J is bisected for the target luminance, as tone is CIE L* of
// Y alone.
func solveLinear(hue, chroma, tone float64) ([3]float64, bool) {
	target := YFromLstar(tone)
	lo, hi := 0.0, 100.0
	for range 40 {
		mid := (lo + hi) / 2
		xyz := xyzFromCAM16(cam16{hue: hue, chroma: chroma, j: mid})
		if xyz[1] < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	xyz := xyzFromCAM16(cam16{hue: hue, chroma: chroma, j: (lo + hi) / 2})
	if math.Abs(xyz[1]-target) > 0.05 {
		// The tone is unreachable at this chroma, as a saturated color in
		// very dark or very light tones is.
		return [3]float64{}, false
	}
	lin := linearRGBFromXYZ(xyz)
	return lin, inGamut(lin)
}

// viewingConditions is the CAM16 viewing environment: Material's default of
// D65, a 50% gray background, average surround and no discounting of the
// illuminant.
type viewingConditions struct {
	rgbD   [3]float64 // adaptation of each cone
	fl     float64    // luminance level adaptation factor
	flRoot float64
	n      float64
	z      float64
	nc     float64
	ncb    float64
	nbb    float64
	c      float64
	aw     float64
}

// defaultViewingConditions is Materials's ViewingConditions.DEFAULT.
var defaultViewingConditions = newViewingConditions(
	[3]float64{whitePointX, whitePointY, whitePointZ},
	200/math.Pi*YFromLstar(50)/100, // adapting luminance
	50,                             // background L*
	2,                              // surround: average
	false,                          // discounting illuminant
)

// newViewingConditions computes the CAM16 viewing conditions. See
// https://doi.org/10.1002/col.22131 and Material's ViewingConditions.
func newViewingConditions(whitePoint [3]float64, adaptingLuminance, backgroundLstar, surround float64, discounting bool) viewingConditions {
	rW := 0.401288*whitePoint[0] + 0.650173*whitePoint[1] - 0.051461*whitePoint[2]
	gW := -0.250268*whitePoint[0] + 1.204414*whitePoint[1] + 0.045854*whitePoint[2]
	bW := -0.002079*whitePoint[0] + 0.048952*whitePoint[1] + 0.953127*whitePoint[2]

	f := 0.8 + surround/10
	// Scale the surround from 0..2 to CAM16's 0.8..1.0, interpolating the
	// exponential non-linearity: 0.525 at 0.8, 0.59 at 0.9 and 0.69 at 1.0.
	c := 0.525 + (0.59-0.525)*(f-0.8)*10
	if f >= 0.9 {
		c = 0.59 + (0.69-0.59)*(f-0.9)*10
	}
	d := 1.0
	if !discounting {
		d = f * (1 - (1.0/3.6)*math.Exp((-adaptingLuminance-42)/92))
		d = min(max(d, 0), 1)
	}
	nc := f
	rgbD := [3]float64{
		d*(100/rW) + 1 - d,
		d*(100/gW) + 1 - d,
		d*(100/bW) + 1 - d,
	}

	k := 1 / (5*adaptingLuminance + 1)
	k4 := k * k * k * k
	k4f := 1 - k4
	fl := k4*adaptingLuminance + 0.1*k4f*k4f*math.Cbrt(5*adaptingLuminance)
	n := YFromLstar(backgroundLstar) / whitePoint[1]
	z := 1.48 + math.Sqrt(n)
	nbb := 0.725 / math.Pow(n, 0.2)
	ncb := nbb

	rgbAFactors := [3]float64{
		math.Pow(fl*rgbD[0]*rW/100, 0.42),
		math.Pow(fl*rgbD[1]*gW/100, 0.42),
		math.Pow(fl*rgbD[2]*bW/100, 0.42),
	}
	rgbA := [3]float64{
		400 * rgbAFactors[0] / (rgbAFactors[0] + 27.13),
		400 * rgbAFactors[1] / (rgbAFactors[1] + 27.13),
		400 * rgbAFactors[2] / (rgbAFactors[2] + 27.13),
	}
	aw := (2*rgbA[0] + rgbA[1] + 0.05*rgbA[2]) * nbb

	return viewingConditions{
		rgbD:   rgbD,
		fl:     fl,
		flRoot: math.Pow(fl, 0.25),
		n:      n,
		z:      z,
		nc:     nc,
		ncb:    ncb,
		nbb:    nbb,
		c:      c,
		aw:     aw,
	}
}

// cam16 is the CAM16 appearance of a color.
type cam16 struct {
	hue    float64
	chroma float64
	j      float64
	q      float64
	m      float64
	s      float64
}

// cam16FromXYZ returns the CAM16 appearance of CIE XYZ under D65 (Y in
// 0..100).
func cam16FromXYZ(xyz [3]float64) cam16 {
	vc := defaultViewingConditions

	rC := 0.401288*xyz[0] + 0.650173*xyz[1] - 0.051461*xyz[2]
	gC := -0.250268*xyz[0] + 1.204414*xyz[1] + 0.045854*xyz[2]
	bC := -0.002079*xyz[0] + 0.048952*xyz[1] + 0.953127*xyz[2]

	rA := adaptResponse(vc.rgbD[0]*rC, vc.fl)
	gA := adaptResponse(vc.rgbD[1]*gC, vc.fl)
	bA := adaptResponse(vc.rgbD[2]*bC, vc.fl)

	a := (11*rA - 12*gA + bA) / 11
	b := (rA + gA - 2*bA) / 9
	u := (20*rA + 20*gA + 21*bA) / 20
	p2 := (40*rA + 20*gA + bA) / 20

	hue := SanitizeDegrees(math.Atan2(b, a) * 180 / math.Pi)
	ac := p2 * vc.nbb
	j := 100 * math.Pow(ac/vc.aw, vc.c*vc.z)
	q := (4 / vc.c) * math.Sqrt(j/100) * (vc.aw + 4) * vc.flRoot

	huePrime := hue
	if huePrime < 20.14 {
		huePrime += 360
	}
	eHue := 0.25 * (math.Cos(huePrime*math.Pi/180+2) + 3.8)
	p1 := 50000.0 / 13.0 * eHue * vc.nc * vc.ncb
	t := p1 * math.Hypot(a, b) / (u + 0.305)
	alpha := math.Pow(1.64-math.Pow(0.29, vc.n), 0.73) * math.Pow(t, 0.9)

	chroma := alpha * math.Sqrt(j/100)
	return cam16{
		hue:    hue,
		chroma: chroma,
		j:      j,
		q:      q,
		m:      chroma * vc.flRoot,
		s:      50 * math.Sqrt(alpha*vc.c/(vc.aw+4)),
	}
}

// xyzFromCAM16 returns the CIE XYZ under D65 (Y in 0..100) of the CAM16
// appearance with lightness j, chroma and hue. It is the inverse of
// cam16FromXYZ, without clamping.
func xyzFromCAM16(c cam16) [3]float64 {
	vc := defaultViewingConditions

	alpha := 0.0
	if c.chroma != 0 && c.j != 0 {
		alpha = c.chroma / math.Sqrt(c.j/100)
	}
	t := math.Pow(alpha/math.Pow(1.64-math.Pow(0.29, vc.n), 0.73), 1.0/0.9)
	hRad := c.hue * math.Pi / 180
	eHue := 0.25 * (math.Cos(hRad+2) + 3.8)
	ac := vc.aw * math.Pow(c.j/100, 1/(vc.c*vc.z))
	p1 := eHue * (50000.0 / 13.0) * vc.nc * vc.ncb
	p2 := ac / vc.nbb
	hSin := math.Sin(hRad)
	hCos := math.Cos(hRad)
	gamma := 23 * (p2 + 0.305) * t / (23*p1 + 11*t*hCos + 108*t*hSin)
	a := gamma * hCos
	b := gamma * hSin

	rA := (460*p2 + 451*a + 288*b) / 1403
	gA := (460*p2 - 891*a - 261*b) / 1403
	bA := (460*p2 - 220*a - 6300*b) / 1403

	rF := inverseAdaptResponse(rA, vc.fl) / vc.rgbD[0]
	gF := inverseAdaptResponse(gA, vc.fl) / vc.rgbD[1]
	bF := inverseAdaptResponse(bA, vc.fl) / vc.rgbD[2]

	return [3]float64{
		1.86206786*rF - 1.01125463*gF + 0.14918677*bF,
		0.38752654*rF + 0.62144744*gF - 0.00897398*bF,
		-0.01584150*rF - 0.03412294*gF + 1.04996444*bF,
	}
}

// adaptResponse is the chromatic adaptation of one cone response.
func adaptResponse(component, fl float64) float64 {
	af := math.Pow(fl*math.Abs(component)/100, 0.42)
	return math.Copysign(400*af/(af+27.13), component)
}

// inverseAdaptResponse is the inverse of adaptResponse.
func inverseAdaptResponse(component, fl float64) float64 {
	// The forward response stays within ±400; a component at that limit
	// would divide by zero, so cap the denominator and let the caller clamp
	// the resulting color into the gamut.
	magnitude := math.Abs(component)
	denominator := 400 - magnitude
	if denominator < 1e-6 {
		denominator = 1e-6
	}
	base := 27.13 * magnitude / denominator
	return math.Copysign(100/fl*math.Pow(base, 1.0/0.42), component)
}
