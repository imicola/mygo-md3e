package docview

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
)

// OutlineItem describes a Markdown heading entry for docview.Outline and
// docview.ExtractOutline.
type OutlineItem struct {
	// Title is the plain heading text with inline formatting stripped.
	Title string
	// Slug is the GitHub-style anchor slug, disambiguated with -1, -2, ...
	// suffixes when duplicate titles appear in the same document.
	Slug string
	// Level is the heading rank (1..6).
	Level int
	// BlockIndex is the index of the heading block inside parser.Document (-1
	// when constructed manually outside ExtractOutline/State).
	BlockIndex int
	// OffsetY is the measured vertical distance in DIP from the top of the
	// rendered Markdown root.
	OffsetY float32
	// HasOffset reports whether OffsetY has been measured from layout.
	HasOffset bool
}

// ExtractOutline parses src using ext (parser.Default when zero or omitted) and
// returns its headings in document order with unique GitHub-style slugs.
func ExtractOutline(src string, ext ...parser.Extensions) []OutlineItem {
	resolved := parser.Default
	if len(ext) > 0 && ext[0] != 0 {
		resolved = ext[0]
	}
	doc := parser.New(resolved)
	doc.Set(src)
	return extractOutlineFromDoc(doc, nil)
}

func extractOutlineFromDoc(doc *parser.Document, dst []OutlineItem) []OutlineItem {
	if doc == nil {
		return dst[:0]
	}
	dst = dst[:0]
	var used map[string]bool
	var nextSuffix map[string]int
	for i := 1; i < doc.Len(); i++ {
		b := doc.Block(i)
		if b.Kind != parser.Heading {
			continue
		}
		title := strings.TrimSpace(parser.PlainText(doc.Inlines(i)))
		if title == "" {
			continue
		}
		base := markdown.Slug(title)
		if base == "" {
			base = fmt.Sprintf("section-%d", len(dst)+1)
		}
		slug := base
		if used == nil {
			used = make(map[string]bool)
		}
		if used[slug] {
			n := max(1, nextSuffix[base])
			for used[slug] {
				slug = fmt.Sprintf("%s-%d", base, n)
				n++
			}
			if nextSuffix == nil {
				nextSuffix = make(map[string]int)
			}
			nextSuffix[base] = n
		}
		used[slug] = true

		dst = append(dst, OutlineItem{
			Title:      title,
			Slug:       slug,
			Level:      b.Level,
			BlockIndex: i,
		})
	}
	return dst
}

// State links a docview.Markdown / docview.MarkdownStream view with a
// docview.Outline navigation sidebar, tracking extracted headings, layout Y
// offsets, active scroll-spy index, and programmatic heading jumps.
type State struct {
	src string
	ext parser.Extensions
	doc *parser.Document

	items       []OutlineItem
	headingTops map[int]float32
	hasOffsets  bool

	scroll        *ui.ScrollState
	anchor        string
	anchorVersion uint64

	pendingJumpIdx int
	hasPendingJump bool
}

// Items returns a copy of the extracted document headings (with measured OffsetY
// values when laid out inside docview.Markdown or docview.MarkdownStream).
func (s *State) Items() []OutlineItem {
	if s == nil || len(s.items) == 0 {
		return nil
	}
	out := make([]OutlineItem, len(s.items))
	copy(out, s.items)
	return out
}

// ActiveIndex returns the index of the currently active heading for scrollY
// (defaulting to the attached ui.ScrollState.Y when scrollY is omitted), or -1
// if there are no headings.
func (s *State) ActiveIndex(scrollY ...float32) int {
	if s == nil || len(s.items) == 0 {
		return -1
	}
	var y, maxY float32
	hasScroll := false
	if len(scrollY) > 0 {
		y = scrollY[0]
		hasScroll = true
	} else if s.scroll != nil {
		y = s.scroll.Y
		maxY = s.scroll.MaxY
		hasScroll = true
	}

	if s.hasPendingJump && s.pendingJumpIdx >= 0 && s.pendingJumpIdx < len(s.items) {
		if s.scroll != nil && s.scroll.Scrolling() {
			return s.pendingJumpIdx
		}
		// Keep explicit selection when no ScrollState is attached or while the
		// jump has not yet moved away from its target.
		if s.scroll == nil {
			return s.pendingJumpIdx
		}
	}

	if !hasScroll || !s.hasOffsets {
		if s.hasPendingJump && s.pendingJumpIdx >= 0 && s.pendingJumpIdx < len(s.items) {
			return s.pendingJumpIdx
		}
		return 0
	}

	if maxY > 0 && y >= maxY-1 {
		return len(s.items) - 1
	}
	if y <= 1 {
		return 0
	}

	const topThreshold = float32(28)
	active := 0
	for i, it := range s.items {
		if it.HasOffset && it.OffsetY <= y+topThreshold {
			active = i
		}
	}
	return active
}

// ActiveSlug returns the Slug of ActiveIndex, or "" when there are no headings.
func (s *State) ActiveSlug() string {
	idx := s.ActiveIndex()
	if s == nil || idx < 0 || idx >= len(s.items) {
		return ""
	}
	return s.items[idx].Slug
}

// JumpTo scrolls the linked Markdown view to the heading matching slug. If no
// heading matches slug, it sets the Markdown anchor directly so ui/markdown can
// still resolve standard heading slugs.
func (s *State) JumpTo(slug string) {
	if s == nil || slug == "" {
		return
	}
	clean := strings.TrimPrefix(slug, "#")
	for i := range s.items {
		if s.items[i].Slug == clean || strings.EqualFold(s.items[i].Slug, clean) {
			s.jumpToIndex(i, ui.Smooth, true)
			return
		}
	}
	s.anchor = clean
	s.hasPendingJump = false
	s.anchorVersion++
}

func (s *State) outlineBlock(doc *parser.Document, slug string) int {
	for _, item := range s.items {
		if strings.EqualFold(item.Slug, slug) && item.BlockIndex < doc.Len() && doc.Block(item.BlockIndex).Kind == parser.Heading {
			return item.BlockIndex
		}
	}
	return -1
}

// JumpToIndex scrolls the linked Markdown view to heading index idx.
func (s *State) JumpToIndex(idx int) {
	if s == nil || idx < 0 || idx >= len(s.items) {
		return
	}
	s.jumpToIndex(idx, ui.Smooth, true)
}

func (s *State) jumpToIndex(idx int, anim ui.Animation, preferScroll bool) {
	if s == nil || idx < 0 || idx >= len(s.items) {
		return
	}
	item := s.items[idx]
	s.pendingJumpIdx = idx
	s.hasPendingJump = true

	if preferScroll && s.scroll != nil && item.HasOffset {
		target := max(float32(0), item.OffsetY-8)
		s.scroll.ScrollTo(s.scroll.X, target, anim)
		return
	}
	s.anchor = item.Slug
	s.anchorVersion++
}

func (s *State) attachScroll(scroll *ui.ScrollState) {
	if s == nil || scroll == nil {
		return
	}
	s.scroll = scroll
	if s.hasPendingJump && !scroll.Scrolling() {
		// Once smooth glide settles, clear pending jump as soon as the user
		// scrolls away from the (possibly MaxY-clamped) target position.
		if s.pendingJumpIdx >= 0 && s.pendingJumpIdx < len(s.items) {
			it := s.items[s.pendingJumpIdx]
			if it.HasOffset {
				clampedTarget := max(float32(0), it.OffsetY-8)
				if scroll.MaxY > 0 && clampedTarget > scroll.MaxY {
					clampedTarget = scroll.MaxY
				}
				if math.Abs(float64(scroll.Y-clampedTarget)) > 12 {
					s.hasPendingJump = false
				}
			}
		}
	}
}

func (s *State) syncDocument(src string, ext parser.Extensions) bool {
	if ext == 0 {
		ext = parser.Default
	}
	if s.doc != nil && s.src == src && s.ext == ext {
		return false
	}
	if s.doc == nil || s.ext != ext {
		s.doc = parser.New(ext)
		s.ext = ext
	}
	prevSrc := s.src
	var prevSlugs []OutlineItem
	if (s.hasPendingJump || s.anchor != "") && len(s.items) > 0 {
		prevSlugs = make([]OutlineItem, len(s.items))
		copy(prevSlugs, s.items)
	}
	s.doc.Set(src)
	s.src = src
	s.hasOffsets = false
	s.items = extractOutlineFromDoc(s.doc, s.items)
	if s.pendingJumpIdx >= len(s.items) ||
		(len(prevSlugs) > 0 && !outlineHeadingsPrefix(prevSlugs, s.items)) ||
		(prevSrc != "" && !strings.HasPrefix(src, prevSrc) && len(prevSlugs) == 0) {
		s.hasPendingJump = false
		s.pendingJumpIdx = -1
		s.anchor = ""
	}
	return true
}

func outlineHeadingsPrefix(prev, next []OutlineItem) bool {
	if len(prev) > len(next) {
		return false
	}
	for i := range prev {
		if prev[i].Slug != next[i].Slug || prev[i].Level != next[i].Level {
			return false
		}
	}
	return true
}

func (s *State) recordHeadingTop(blockIdx int, y float32) {
	if s == nil || blockIdx <= 0 {
		return
	}
	if s.headingTops == nil {
		s.headingTops = map[int]float32{}
	}
	s.headingTops[blockIdx] = y
}

func (s *State) updateOffsets(rootY float32) bool {
	prevActive := s.ActiveIndex()
	s.hasOffsets = false
	for i := range s.items {
		item := &s.items[i]
		top, ok := s.headingTops[item.BlockIndex]
		item.HasOffset = ok
		item.OffsetY = 0
		if ok {
			item.OffsetY = max(float32(0), top-rootY)
			s.hasOffsets = true
		}
	}
	return s.ActiveIndex() != prevActive
}

func jumpAnimationFor(th *md3e.Theme, c *ui.Context) ui.Animation {
	if th == nil {
		th = md3e.CurrentTheme(c)
	}
	reduce := th.ReduceMotion || (c != nil && c.Preferences().ReduceMotion)
	return motion.Resolve(th.Motion, motion.DefaultSpatial, reduce)
}
