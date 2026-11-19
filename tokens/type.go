package tokens

// TypeRole is one of the 15 roles of the MD3 type scale.
type TypeRole int

const (
	DisplayLarge TypeRole = iota
	DisplayMedium
	DisplaySmall
	HeadlineLarge
	HeadlineMedium
	HeadlineSmall
	TitleLarge
	TitleMedium
	TitleSmall
	BodyLarge
	BodyMedium
	BodySmall
	LabelLarge
	LabelMedium
	LabelSmall
)

// TypeRoleCount is the number of roles in the type scale.
const TypeRoleCount = 15

// TypeStyle is one role of the type scale: size and line height in DIP,
// weight and letter spacing in DIP.
type TypeStyle struct {
	Size       float64
	LineHeight float64
	Weight     int
	Tracking   float64
}

// TypographyScale is the MD3 type scale in its standard and emphasized forms.
// Both forms share their sizes, line heights and tracking; emphasized roles
// carry more weight.
type TypographyScale struct {
	Standard   [TypeRoleCount]TypeStyle
	Emphasized [TypeRoleCount]TypeStyle
}

// NewTypographyScale returns the MD3 type scale.
func NewTypographyScale() TypographyScale {
	return TypographyScale{
		Standard:   standardType,
		Emphasized: emphasizedType,
	}
}

// Style returns the style of role, in its emphasized form when emphasized is
// true. An unknown role returns the zero TypeStyle.
func (t *TypographyScale) Style(role TypeRole, emphasized bool) TypeStyle {
	if role < 0 || role >= TypeRoleCount {
		return TypeStyle{}
	}
	if emphasized {
		return t.Emphasized[role]
	}
	return t.Standard[role]
}

var standardType = [TypeRoleCount]TypeStyle{
	DisplayLarge:   {Size: 57, LineHeight: 64, Weight: 400, Tracking: -0.25},
	DisplayMedium:  {Size: 45, LineHeight: 52, Weight: 400, Tracking: 0},
	DisplaySmall:   {Size: 36, LineHeight: 44, Weight: 400, Tracking: 0},
	HeadlineLarge:  {Size: 32, LineHeight: 40, Weight: 400, Tracking: 0},
	HeadlineMedium: {Size: 28, LineHeight: 36, Weight: 400, Tracking: 0},
	HeadlineSmall:  {Size: 24, LineHeight: 32, Weight: 400, Tracking: 0},
	TitleLarge:     {Size: 22, LineHeight: 28, Weight: 400, Tracking: 0},
	TitleMedium:    {Size: 16, LineHeight: 24, Weight: 500, Tracking: 0.15},
	TitleSmall:     {Size: 14, LineHeight: 20, Weight: 500, Tracking: 0.1},
	BodyLarge:      {Size: 16, LineHeight: 24, Weight: 400, Tracking: 0.5},
	BodyMedium:     {Size: 14, LineHeight: 20, Weight: 400, Tracking: 0.25},
	BodySmall:      {Size: 12, LineHeight: 16, Weight: 400, Tracking: 0.4},
	LabelLarge:     {Size: 14, LineHeight: 20, Weight: 500, Tracking: 0.1},
	LabelMedium:    {Size: 12, LineHeight: 16, Weight: 500, Tracking: 0.5},
	LabelSmall:     {Size: 11, LineHeight: 16, Weight: 500, Tracking: 0.5},
}

var emphasizedType = [TypeRoleCount]TypeStyle{
	DisplayLarge:   {Size: 57, LineHeight: 64, Weight: 500, Tracking: -0.25},
	DisplayMedium:  {Size: 45, LineHeight: 52, Weight: 500, Tracking: 0},
	DisplaySmall:   {Size: 36, LineHeight: 44, Weight: 500, Tracking: 0},
	HeadlineLarge:  {Size: 32, LineHeight: 40, Weight: 500, Tracking: 0},
	HeadlineMedium: {Size: 28, LineHeight: 36, Weight: 500, Tracking: 0},
	HeadlineSmall:  {Size: 24, LineHeight: 32, Weight: 500, Tracking: 0},
	TitleLarge:     {Size: 22, LineHeight: 28, Weight: 500, Tracking: 0},
	TitleMedium:    {Size: 16, LineHeight: 24, Weight: 600, Tracking: 0.15},
	TitleSmall:     {Size: 14, LineHeight: 20, Weight: 600, Tracking: 0.1},
	BodyLarge:      {Size: 16, LineHeight: 24, Weight: 500, Tracking: 0.5},
	BodyMedium:     {Size: 14, LineHeight: 20, Weight: 500, Tracking: 0.25},
	BodySmall:      {Size: 12, LineHeight: 16, Weight: 500, Tracking: 0.4},
	LabelLarge:     {Size: 14, LineHeight: 20, Weight: 600, Tracking: 0.1},
	LabelMedium:    {Size: 12, LineHeight: 16, Weight: 600, Tracking: 0.5},
	LabelSmall:     {Size: 11, LineHeight: 16, Weight: 600, Tracking: 0.5},
}
