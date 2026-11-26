package main

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func checkDocsNamedControls(t *testing.T, tt *ui.Tester, where string) {
	t.Helper()
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleToggleButton, ui.RoleTab, ui.RoleRadio,
			ui.RoleCheckBox, ui.RoleSwitch, ui.RoleSlider, ui.RoleTextField,
			ui.RoleMeter, ui.RoleTree, ui.RoleTreeItem:
			if info.Name == "" {
				t.Fatalf("%s: a %v at %v has no accessible name", where, info.Role, info.Bounds)
			}
		}
	}
}

func openDocsWorkbench(t *testing.T, pauseMonitor bool) (*appState, *ui.Tester) {
	t.Helper()
	a, tt := newWorkbenchTester(t, pauseMonitor)

	if err := tt.Click("Docs"); err != nil {
		t.Fatalf("navigate to Docs page: %v", err)
	}
	tt.Advance(900 * time.Millisecond)
	if a.page != 2 {
		t.Fatalf("a.page = %d after clicking Docs, want 2", a.page)
	}
	if !tt.HasText("Knowledge Base") || !tt.HasText("Getting Started with MyGo MD3E") {
		t.Fatal("Docs workbench initial panes did not render")
	}
	return a, tt
}

func TestWorkbenchDocsNavigationAndThreePaneLayout(t *testing.T) {
	a, tt := openDocsWorkbench(t, false)

	if !tt.HasText("MyGo MD3E Workbench — Docs") {
		t.Fatal("WindowTitleBar did not update to Docs title")
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Knowledge base directory"); !ok {
		t.Fatal("TreeView directory missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Document path"); !ok {
		t.Fatal("Breadcrumb document path missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Document outline"); !ok {
		t.Fatal("docview.Outline landmark missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Note"); !ok {
		t.Fatal("initial Getting Started doc missing Note Callout")
	}
	if tt.HasText("Callout & Code Matrix.md") {
		t.Fatal("collapsed Reference folder unexpectedly rendered children initially")
	}
	if tt.FramePending() {
		t.Fatal("idle Docs workbench should settle to FramePending() == false")
	}
	checkDocsNamedControls(t, tt, "docs workbench")

	// Switching back to Monitor resumes live telemetry cleanly.
	if err := tt.Click("Monitor"); err != nil {
		t.Fatalf("navigate back to Monitor: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	if a.page != 0 || !tt.HasText("MyGo MD3E Workbench — Monitor") {
		t.Fatalf("did not return cleanly to Monitor page: page=%d", a.page)
	}
	ticksBefore := a.mon.tickCount
	tt.Advance(1100 * time.Millisecond)
	if a.mon.tickCount <= ticksBefore {
		t.Fatalf("Monitor live telemetry did not resume upon returning: before=%d after=%d", ticksBefore, a.mon.tickCount)
	}
}

func TestWorkbenchDocsTreeViewFilterAndLinks(t *testing.T) {
	a, tt := openDocsWorkbench(t, true)

	// Expand collapsed Reference folder and open Callout & Code Matrix.md.
	refItem, ok := tt.AccessByRole(ui.RoleTreeItem, "Reference")
	if !ok || refItem.Expanded {
		t.Fatalf("Reference tree item missing or unexpectedly expanded: ok=%v expanded=%v", ok, refItem.Expanded)
	}
	tt.ClickAt(centerOf(refItem.Bounds))
	tt.Advance(500 * time.Millisecond)
	if !tt.HasText("Callout & Code Matrix.md") {
		t.Fatal("clicking Reference folder did not expand children")
	}

	if err := tt.Click("Callout & Code Matrix.md"); err != nil {
		t.Fatalf("open Callout & Code Matrix.md: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	for _, kind := range []string{"Note", "Tip", "Important", "Warning", "Caution"} {
		if _, ok := tt.AccessByRole(ui.RoleRegion, kind); !ok {
			t.Fatalf("Callout & Code Matrix missing %s callout region", kind)
		}
	}

	// Open Quick Tip.md (no headings) and verify Outline EmptyText fallback.
	if err := tt.Click("Quick Tip.md"); err != nil {
		t.Fatalf("open Quick Tip.md: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !tt.HasText("No headings in this document.") {
		t.Fatal("Outline did not render EmptyText for heading-less document")
	}

	// Return to Getting Started.md, scroll to the TIP link, and test internal doc:// link jump.
	if err := tt.Click("Getting Started.md"); err != nil {
		t.Fatalf("return to Getting Started.md: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if err := tt.Click("Minimal Application Setup"); err != nil {
		t.Fatalf("jump to Minimal Application Setup in Outline: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	tt.Wheel(540, 380, 0, 500)
	tt.Advance(250 * time.Millisecond)
	if err := tt.Click("Motion & Springs"); err != nil {
		t.Fatalf("click internal doc link Motion & Springs: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if a.docs.activeDocID != "tokens/motion-physics" || !tt.HasText("Expressive Spring Physics") {
		t.Fatalf("internal doc:// link did not navigate: activeDocID=%q", a.docs.activeDocID)
	}

	// Test TreeView live search filter -> empty state -> Clear filter.
	searchField, ok := tt.AccessByRole(ui.RoleTextField, "Filter docs")
	if !ok {
		t.Fatal("Filter docs search field missing")
	}
	tt.ClickAt(centerOf(searchField.Bounds))
	tt.Type("Zero-Bloat")
	tt.Advance(250 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "Zero-Bloat & DCE.md"); !ok {
		t.Fatal("TreeView search filter did not show Zero-Bloat & DCE.md")
	}
	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "Streaming Brief.md"); ok {
		t.Fatal("TreeView search filter did not hide Streaming Brief.md")
	}
	if err := tt.Click("Clear docs filter"); err != nil {
		t.Fatalf("click Clear docs filter: %v", err)
	}
	tt.Advance(250 * time.Millisecond)

	searchField, _ = tt.AccessByRole(ui.RoleTextField, "Filter docs")
	tt.ClickAt(centerOf(searchField.Bounds))
	tt.Type("nonexistent-topic-xyz")
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("No matching documents") {
		t.Fatal("empty tree filter did not show EmptyState")
	}
	if err := tt.Click("Clear filter"); err != nil {
		t.Fatalf("click Clear filter in tree EmptyState: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("Getting Started.md") {
		t.Fatal("Clear filter did not restore Knowledge Base tree")
	}

	// Collapse all and Expand all folders.
	if err := tt.Click("Collapse all folders"); err != nil {
		t.Fatalf("click Collapse all folders: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "Getting Started.md"); ok {
		t.Fatal("Collapse all folders left Getting Started.md visible")
	}
	if err := tt.Click("Expand all folders"); err != nil {
		t.Fatalf("click Expand all folders: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok1 := tt.AccessByRole(ui.RoleTreeItem, "Getting Started.md"); !ok1 {
		t.Fatal("Expand all folders did not reopen Getting Started.md")
	}
	if _, ok2 := tt.AccessByRole(ui.RoleTreeItem, "Callout & Code Matrix.md"); !ok2 {
		t.Fatal("Expand all folders did not reopen Callout & Code Matrix.md")
	}
}

func TestWorkbenchDocsCopyTasksSourceAndOutlineSync(t *testing.T) {
	a, tt := openDocsWorkbench(t, true)

	// Scroll down slightly so the Go code block and checklist are in view.
	tt.Wheel(540, 380, 0, 340)
	tt.Advance(250 * time.Millisecond)

	// Copy code block via header button and verify clipboard + Snackbar toast.
	if err := tt.Click("Copy go code"); err != nil {
		t.Fatalf("click Copy go code: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if !strings.Contains(tt.Clipboard(), "func appView") {
		t.Fatalf("clipboard did not receive go code block: %q", tt.Clipboard())
	}
	if !tt.HasText("Copied go code") {
		t.Fatal("CodeBlock copy did not trigger Snackbar confirmation")
	}

	// Jump via Outline to Interactive Checklist (keeps tasks clear of bottom snackbar) and toggle a checkbox live.
	if err := tt.Click("Interactive Checklist"); err != nil {
		t.Fatalf("jump to Interactive Checklist in Outline: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	taskBox, ok := tt.AccessByRole(ui.RoleCheckBox, "Enable live MarkdownStream playback in the toolbar")
	if !ok || taskBox.Checked {
		t.Fatalf("task checkbox missing or unexpectedly checked: ok=%v checked=%v", ok, taskBox.Checked)
	}
	tt.ClickAt(centerOf(taskBox.Bounds))
	tt.Advance(250 * time.Millisecond)
	taskBox, _ = tt.AccessByRole(ui.RoleCheckBox, "Enable live MarkdownStream playback in the toolbar")
	if !taskBox.Checked || !a.docs.activeDoc().edited {
		t.Fatalf("clicking task checkbox did not update Markdown source: checked=%v edited=%v", taskBox.Checked, a.docs.activeDoc().edited)
	}
	if !tt.HasText("Edited") {
		t.Fatal("Edited status pill missing after task toggle")
	}

	if err := tt.Click("Reset tasks"); err != nil {
		t.Fatalf("click Reset tasks: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if a.docs.activeDoc().edited {
		t.Fatal("Reset tasks did not restore original document content")
	}

	// Toggle Source view (line-numbered docview.CodeBlock) and back.
	if err := tt.Click("Source"); err != nil {
		t.Fatalf("toggle Source chip on: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if !a.docs.showRawSource || !tt.HasText("Getting Started.md") {
		t.Fatal("Source mode did not render raw Markdown CodeBlock")
	}
	if err := tt.Click("Source"); err != nil {
		t.Fatalf("toggle Source chip off: %v", err)
	}
	tt.Advance(250 * time.Millisecond)

	// Outline scroll-spy jump to last section and return to Top.
	if err := tt.Click("Keyboard & Selection"); err != nil {
		t.Fatalf("click Keyboard & Selection in Outline: %v", err)
	}
	for range 40 {
		tt.Advance(40 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if a.docs.docScroll.Y <= 0 {
		t.Fatalf("Outline click did not scroll document: scroll.Y=%.1f", a.docs.docScroll.Y)
	}
	if got := a.docs.docState.ActiveSlug(); got != "keyboard--selection" {
		t.Fatalf("ActiveSlug after jump = %q, want keyboard--selection", got)
	}

	if err := tt.Click("Top"); err != nil {
		t.Fatalf("click Top button in Outline footer: %v", err)
	}
	for range 40 {
		tt.Advance(40 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if a.docs.docScroll.Y > 1 {
		t.Fatalf("Top button did not scroll back to top: scroll.Y=%.1f", a.docs.docScroll.Y)
	}

	// Advance past snackbars & animations and verify zero idle frames.
	tt.Advance(4500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Docs workbench left pending frames after settling")
	}
}

func TestWorkbenchDocsStreamingOutlineAndReduceMotion(t *testing.T) {
	a, tt := openDocsWorkbench(t, true)

	if err := tt.Click("Streaming Brief.md"); err != nil {
		t.Fatalf("open Streaming Brief.md: %v", err)
	}
	tt.Advance(300 * time.Millisecond)

	// Outline depth filter H1-H2 hides H3 "Chunk Healing Guarantees", All restores it.
	if _, ok := tt.AccessByRole(ui.RoleButton, "Chunk Healing Guarantees"); !ok {
		t.Fatal("H3 heading button missing in Outline All depth mode")
	}
	if err := tt.Click("H1–H2"); err != nil {
		t.Fatalf("switch Outline depth to H1-H2: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Chunk Healing Guarantees"); ok {
		t.Fatal("H1-H2 depth filter did not hide H3 heading in Outline")
	}
	if err := tt.Click("All"); err != nil {
		t.Fatalf("switch Outline depth back to All: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Chunk Healing Guarantees"); !ok {
		t.Fatal("All depth mode did not restore H3 heading in Outline")
	}

	// Trigger live MarkdownStream playback.
	if err := tt.Click("Stream document"); err != nil {
		t.Fatalf("start MarkdownStream playback: %v", err)
	}
	tt.Advance(150 * time.Millisecond)
	if !a.docs.streamFeeding {
		t.Fatal("expected streamFeeding=true mid-stream")
	}
	if !tt.HasText("Streaming...") || !tt.HasText("AI Architecture Brief") {
		t.Fatal("streaming UI or initial heading missing mid-stream")
	}
	if tt.HasText("Verification Checklist") {
		t.Fatal("entire document appeared before stream finished")
	}

	// Advance virtual clock until stream finishes and settles cleanly.
	for range 90 {
		tt.Advance(50 * time.Millisecond)
		if !a.docs.streamFeeding && !tt.FramePending() {
			break
		}
	}
	if a.docs.streamFeeding || tt.FramePending() {
		t.Fatalf("MarkdownStream did not settle: feeding=%v pending=%v offset=%d/%d",
			a.docs.streamFeeding, tt.FramePending(), a.docs.streamOffset, len(a.docs.activeDoc().content))
	}
	if !tt.HasText("Verification Checklist") {
		t.Fatal("completed MarkdownStream missing final Verification Checklist section")
	}

	// Test ReduceMotion immediate reveal.
	a.reduce = true
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if err := tt.Click("Stream document"); err != nil {
		t.Fatalf("trigger stream under ReduceMotion: %v", err)
	}
	tt.Frame()
	if a.docs.streamFeeding || tt.FramePending() {
		t.Fatalf("ReduceMotion stream did not reveal immediately: feeding=%v pending=%v", a.docs.streamFeeding, tt.FramePending())
	}
	if !tt.HasText("Verification Checklist") {
		t.Fatal("ReduceMotion stream missing full document content")
	}
	tt.Advance(4500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion Docs left pending frames after snackbar dismissed")
	}
}

func TestWorkbenchDocsBreakpointsAndRTL(t *testing.T) {
	a, tt := openDocsWorkbench(t, true)

	// 1. Expanded RTL: Outline on the left, Markdown center, TreeView on the right.
	a.rtl = true
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	outlineNav, okO := tt.AccessByRole(ui.RoleNavigation, "Document outline")
	treeEl, okT := tt.AccessByRole(ui.RoleTree, "Knowledge base directory")
	headingEl, okH := tt.AccessByRole(ui.RoleHeading, "Getting Started with MyGo MD3E")
	if !okO || !okT || !okH {
		t.Fatalf("RTL layout elements missing: outline=%v tree=%v heading=%v", okO, okT, okH)
	}
	if !(outlineNav.Bounds.X < headingEl.Bounds.X && headingEl.Bounds.X < treeEl.Bounds.X) {
		t.Fatalf("RTL 3-pane order invalid: outline.X=%.1f heading.X=%.1f tree.X=%.1f",
			outlineNav.Bounds.X, headingEl.Bounds.X, treeEl.Bounds.X)
	}
	a.rtl = false
	tt.Frame()
	tt.Advance(200 * time.Millisecond)

	// 2. Medium (720x760): TreeView + Markdown visible; Outline collapsed until toggled.
	tt.SetSize(720, 760)
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleTree, "Knowledge base directory"); !ok {
		t.Fatal("Medium layout should keep TreeView directory visible")
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Document outline"); ok {
		t.Fatal("Medium layout should collapse Outline by default to preserve reader width")
	}
	if err := tt.Click("Outline (5)"); err != nil {
		t.Fatalf("toggle Outline visible on Medium: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Document outline"); !ok {
		t.Fatal("Outline chip did not reveal Outline on Medium")
	}

	// 3. Compact (480x760): switches cleanly between Reader, Files (TreeView), and Outline.
	tt.SetSize(480, 760)
	tt.Advance(400 * time.Millisecond)
	if !tt.HasText("Getting Started with MyGo MD3E") {
		t.Fatal("Compact layout should start on document reader")
	}
	if err := tt.Click("Files"); err != nil {
		t.Fatalf("click Files chip on Compact: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleTree, "Knowledge base directory"); !ok {
		t.Fatal("Files chip did not open TreeView on Compact")
	}
	if err := tt.Click("Adaptive Shell.md"); err != nil {
		t.Fatalf("select Adaptive Shell.md on Compact: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !tt.HasText("Adaptive Multi-Pane Layouts") {
		t.Fatal("selecting file in Compact TreeView did not return to reader")
	}
	if err := tt.Click("Outline (4)"); err != nil {
		t.Fatalf("open Outline on Compact: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Document outline"); !ok {
		t.Fatal("Outline chip did not switch Compact view to Outline")
	}
	if err := tt.Click("Back to document"); err != nil {
		t.Fatalf("click Back to document from Outline on Compact: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Adaptive Multi-Pane Layouts") {
		t.Fatal("Back to document did not restore Compact reader")
	}
	checkDocsNamedControls(t, tt, "docs compact")
}
