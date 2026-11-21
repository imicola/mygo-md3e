package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	myshape "mygo-md3e/shape"
)

// galleryTester builds the gallery in a tall window so every control of a page
// is laid out and clickable, and turns reduce motion on so animations settle
// and the smoke checks see a still frame.
func galleryTester(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	a := newApp()
	tt := ui.NewTester(a.view, 1360, 2200)
	tt.Frame()
	if !tt.HasText("MyGo MD3E Gallery") {
		t.Fatal("the gallery title is missing")
	}
	if err := tt.Click("Reduce motion"); err != nil {
		t.Fatalf("click Reduce motion: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if !a.reduce {
		t.Fatal("the Reduce motion check box did not turn the setting on")
	}
	return a, tt
}

// gotoRoute shows a page the way the rail does and waits for the transition.
func gotoRoute(t *testing.T, a *app, tt *ui.Tester, path string) {
	t.Helper()
	a.router.Push(path)
	tt.Advance(600 * time.Millisecond)
	if got := a.router.Path(); got != path {
		t.Fatalf("the router shows %q, want %q", got, path)
	}
	if tt.FramePending() {
		t.Fatalf("%s keeps asking for frames after its transition", path)
	}
}

// checkNamed fails on an interactive element assistive technology cannot name.
func checkNamed(t *testing.T, tt *ui.Tester, where string) {
	t.Helper()
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleTab, ui.RoleRadio, ui.RoleCheckBox, ui.RoleSwitch,
			ui.RoleSlider, ui.RoleTextField:
			if info.Name == "" {
				t.Fatalf("%s: a %v at %v has no name", where, info.Role, info.Bounds)
			}
		}
	}
}

// pageMarkers are texts only the matching page shows, so a route check cannot
// pass on the rail's own labels.
var pageMarkers = map[string]string{
	"/theme":      "Dynamic color roles",
	"/shapes":     "Shape catalog",
	"/motion":     "Progress console",
	"/actions":    "FAB family",
	"/inputs":     "Chips",
	"/feedback":   "Snackbars",
	"/navigation": "Carousels",
	"/overlays":   "Time pickers",
}

// TestGalleryRoutesClickThrough switches every page from the rail, checks its
// marker, the absence of unnamed controls and that the page settles.
func TestGalleryRoutesClickThrough(t *testing.T) {
	a, tt := galleryTester(t)
	for _, route := range galleryRoutes {
		if err := tt.Click(route.label); err != nil {
			t.Fatalf("click the rail's %q: %v", route.label, err)
		}
		tt.Advance(600 * time.Millisecond)
		if got := a.router.Path(); got != route.path {
			t.Fatalf("the rail's %q shows %q, want %q", route.label, got, route.path)
		}
		if marker := pageMarkers[route.path]; !tt.HasText(marker) {
			t.Fatalf("%s does not show %q", route.path, marker)
		}
		checkNamed(t, tt, route.path)
		if tt.FramePending() {
			t.Fatalf("%s keeps asking for frames", route.path)
		}
	}
}

// TestGalleryThemeControls checks the live seed, variant, contrast, light and
// dark, motion scheme, reduce motion and right-to-left controls.
func TestGalleryThemeControls(t *testing.T) {
	a, tt := galleryTester(t)

	if err := tt.Click("Dark"); err != nil {
		t.Fatalf("click Dark: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if a.darkIdx != 1 {
		t.Fatalf("the appearance index is %d, want 1", a.darkIdx)
	}
	if err := tt.Click("High"); err != nil {
		t.Fatalf("click High contrast: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.contrast != 2 {
		t.Fatalf("the contrast index is %d, want 2", a.contrast)
	}
	if err := tt.Click("Seed Teal"); err != nil {
		t.Fatalf("click Seed Teal: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.seedIdx != 5 {
		t.Fatalf("the seed index is %d, want 5", a.seedIdx)
	}

	// The variant menu is a DropdownMenu of radio rows.
	if err := tt.Click("Variant: TonalSpot"); err != nil {
		t.Fatalf("open the variant menu: %v", err)
	}
	tt.Advance(100 * time.Millisecond)
	if err := tt.Click("Vibrant"); err != nil {
		t.Fatalf("choose Vibrant: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.variantIdx != 2 {
		t.Fatalf("the variant index is %d, want 2", a.variantIdx)
	}

	if err := tt.Click("Standard"); err != nil {
		t.Fatalf("click Standard motion: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.motionIdx != 1 {
		t.Fatalf("the motion scheme index is %d, want 1", a.motionIdx)
	}

	// Right to left lays the whole shell out from the right, overlays included.
	if err := tt.Click("RTL"); err != nil {
		t.Fatalf("click RTL: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !a.rtl {
		t.Fatal("the RTL switch did not turn on")
	}
	rail, ok := tt.AccessByRole(ui.RoleNavigation, "Gallery pages")
	if !ok {
		t.Fatal("the gallery rail is not a named navigation landmark")
	}
	if rail.Bounds.X < 700 {
		t.Fatalf("right to left puts the rail at %.1f, not at the window's right", rail.Bounds.X)
	}
	checkNamed(t, tt, "right to left")
	if tt.FramePending() {
		t.Fatal("the right-to-left gallery keeps asking for frames")
	}
	// Every page lays out right to left with named controls and settles.
	for _, route := range galleryRoutes {
		gotoRoute(t, a, tt, route.path)
		if !tt.HasText(pageMarkers[route.path]) {
			t.Fatalf("%s does not show its marker right to left", route.path)
		}
		checkNamed(t, tt, route.path+" right to left")
	}
	gotoRoute(t, a, tt, "/theme")
	if err := tt.Click("RTL"); err != nil {
		t.Fatalf("click RTL again: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if a.rtl {
		t.Fatal("the RTL switch did not turn off")
	}
}

// TestGalleryShapesAndActions exercises the shape playground and the action
// demos.
func TestGalleryShapesAndActions(t *testing.T) {
	a, tt := galleryTester(t)
	gotoRoute(t, a, tt, "/shapes")

	if err := tt.Click("Morph to Heart"); err != nil {
		t.Fatalf("click Morph to Heart: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	heart, ok := myshape.ByName("Heart")
	if !ok {
		t.Fatal("Heart is not in the catalog")
	}
	if int(a.shapes.b) != indexOfShape(a, heart.Name) {
		t.Fatalf("the morph's end is %v, want Heart", a.shapes.b)
	}
	if a.shapes.morph == nil {
		t.Fatal("the morph playground built no morph")
	}
	if err := tt.Click("Reverse"); err != nil {
		t.Fatalf("click Reverse: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if int(a.shapes.a) != indexOfShape(a, heart.Name) || int(a.shapes.b) != 0 {
		t.Fatalf("reversing made the ends %v and %v", a.shapes.a, a.shapes.b)
	}
	checkNamed(t, tt, "/shapes")

	gotoRoute(t, a, tt, "/actions")
	if err := tt.Click("Toggle favorite"); err != nil {
		t.Fatalf("click Toggle favorite: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if !a.actions.favorite {
		t.Fatal("the toggle icon button did not turn on")
	}
	if err := tt.Click("Filled S"); err != nil {
		t.Fatalf("click Filled S: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.actions.lastAction != "Filled S" {
		t.Fatalf("the last action is %q, want Filled S", a.actions.lastAction)
	}
	if err := tt.Click("Create"); err != nil {
		t.Fatalf("open the FAB menu: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("Add photo"); err != nil {
		t.Fatalf("choose Add photo: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.actions.lastAction != "Add photo" {
		t.Fatalf("the FAB menu's action is %q, want Add photo", a.actions.lastAction)
	}
	checkNamed(t, tt, "/actions")
	if tt.FramePending() {
		t.Fatal("/actions keeps asking for frames")
	}
}

// TestGalleryInputsAndFeedback exercises the selection controls, chips, text
// fields, snackbars and lists.
func TestGalleryInputsAndFeedback(t *testing.T) {
	a, tt := galleryTester(t)
	gotoRoute(t, a, tt, "/inputs")

	if a.inputs.checked != true {
		t.Fatal("the Wi-Fi check box does not start checked")
	}
	if err := tt.Click("Wi-Fi"); err != nil {
		t.Fatalf("click Wi-Fi: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.inputs.checked {
		t.Fatal("the Wi-Fi check box did not turn off")
	}
	if err := tt.Click("Spacious"); err != nil {
		t.Fatalf("click Spacious: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.inputs.radio != 2 {
		t.Fatalf("the radio chose %d, want 2", a.inputs.radio)
	}
	if err := tt.Click("Remove Kotlin"); err != nil {
		t.Fatalf("remove the Kotlin chip: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if len(a.inputs.tags) != 2 {
		t.Fatalf("the chips are %v, want two left", a.inputs.tags)
	}
	if err := tt.Click("Reset chips"); err != nil {
		t.Fatalf("reset the chips: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if len(a.inputs.tags) != 3 {
		t.Fatalf("the chips are %v, want three", a.inputs.tags)
	}
	field, ok := tt.AccessByRole(ui.RoleTextField, "Full name")
	if !ok {
		t.Fatal("the Full name field is missing")
	}
	tt.ClickAt(centerOf(field.Bounds))
	tt.Type("Ada Lovelace")
	tt.Advance(200 * time.Millisecond)
	if a.inputs.name != "Ada Lovelace" {
		t.Fatalf("the name field holds %q", a.inputs.name)
	}
	checkNamed(t, tt, "/inputs")

	gotoRoute(t, a, tt, "/feedback")
	if err := tt.Click("Show snackbar"); err != nil {
		t.Fatalf("click Show snackbar: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Note archived") {
		t.Fatal("the snackbar did not show")
	}
	tt.Advance(5 * time.Second)
	if tt.HasText("Note archived") {
		t.Fatal("the snackbar did not go by itself")
	}
	if err := tt.Click("Clickable card"); err != nil {
		t.Fatalf("click the clickable card: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.feedback.clicked != "Clickable card" {
		t.Fatalf("the card click reads %q", a.feedback.clicked)
	}
	if err := tt.Click("Starred"); err != nil {
		t.Fatalf("click the Starred row: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.feedback.segment != 1 {
		t.Fatalf("the list chose row %d, want 1", a.feedback.segment)
	}
	checkNamed(t, tt, "/feedback")
	if tt.FramePending() {
		t.Fatal("/feedback keeps asking for frames")
	}
}

// TestGalleryNavigation exercises the navigation bar, drawer, secondary tabs
// and a carousel.
func TestGalleryNavigation(t *testing.T) {
	a, tt := galleryTester(t)
	gotoRoute(t, a, tt, "/navigation")

	if err := tt.Click("Library"); err != nil {
		t.Fatalf("click Library: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.navigation.bar != 3 {
		t.Fatalf("the navigation bar chose %d, want 3", a.navigation.bar)
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatalf("click Settings in the drawer: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.navigation.drawer != 3 {
		t.Fatalf("the drawer chose %d, want 3", a.navigation.drawer)
	}
	if err := tt.Click("Files"); err != nil {
		t.Fatalf("click the Files tab: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.navigation.secondary != 1 {
		t.Fatalf("the secondary tabs chose %d, want 1", a.navigation.secondary)
	}
	if err := tt.Click("Lakes"); err != nil {
		t.Fatalf("click the Lakes slide: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.navigation.hero != 1 {
		t.Fatalf("the hero carousel chose %d, want 1", a.navigation.hero)
	}
	checkNamed(t, tt, "/navigation")
	if tt.FramePending() {
		t.Fatal("/navigation keeps asking for frames")
	}
}

// TestGalleryOverlays opens and closes the dialogs, sheets and modal pickers,
// and uses the menus.
func TestGalleryOverlays(t *testing.T) {
	a, tt := galleryTester(t)
	gotoRoute(t, a, tt, "/overlays")

	if err := tt.Click("Open alert dialog"); err != nil {
		t.Fatalf("open the alert dialog: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Save document"); !ok {
		t.Fatal("the alert dialog is not an alert dialog named Save document")
	}
	if err := tt.Click("Not now"); err != nil {
		t.Fatalf("dismiss the alert dialog: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Save document"); ok {
		t.Fatal("the alert dialog stayed open")
	}

	if err := tt.Click("Open full-screen dialog"); err != nil {
		t.Fatalf("open the full-screen dialog: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "New document"); !ok {
		t.Fatal("the full-screen dialog is not a dialog named New document")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(600 * time.Millisecond)
	if a.overlays.full {
		t.Fatal("Escape did not close the full-screen dialog")
	}

	if err := tt.Click("Open modal sheet"); err != nil {
		t.Fatalf("open the modal sheet: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); !ok {
		t.Fatal("the bottom sheet is not a dialog named Share sheet")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(600 * time.Millisecond)
	if a.overlays.sheet {
		t.Fatal("Escape did not close the bottom sheet")
	}

	if err := tt.Click("Open modal date picker"); err != nil {
		t.Fatalf("open the modal date picker: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Modal due date"); !ok {
		t.Fatal("the modal date picker is not a dialog named Modal due date")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(600 * time.Millisecond)
	if a.overlays.dateModal {
		t.Fatal("Escape did not close the modal date picker")
	}

	if err := tt.Click("Dropdown menu"); err != nil {
		t.Fatalf("open the dropdown menu: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if err := tt.Click("Rename"); err != nil {
		t.Fatalf("choose Rename: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.overlays.last != "Rename" {
		t.Fatalf("the menu chose %q, want Rename", a.overlays.last)
	}
	checkNamed(t, tt, "/overlays")
	if tt.FramePending() {
		t.Fatal("/overlays keeps asking for frames")
	}
}

// indexOfShape is the catalog index of a shape name.
func indexOfShape(a *app, name string) int {
	for i, s := range a.shapes.all {
		if s.Name == name {
			return i
		}
	}
	return -1
}

// centerOf is the middle of a rect.
func centerOf(r ui.Rect) (float32, float32) {
	return r.X + r.W/2, r.Y + r.H/2
}
