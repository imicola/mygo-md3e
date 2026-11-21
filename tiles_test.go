package md3e

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestSwitchCardRowClickSwitchClickKeyboardAndKeyIsolation(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	wifi := false
	bt := true
	wifiChanges := 0
	swapped := false

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(380, 220).Background(canvasRGB).Padding(16).Gap(10).Children(func() {
				renderWifi := func() {
					SwitchCard(c.Key("wifi-card"), "Wi-Fi", &wifi, SwitchCardOptions{
						Subtitle:    "Connect automatically",
						LeadingIcon: mdicon.Star.Draw,
						Hairline:    true,
						OnChanged:   func(on bool) { wifiChanges++ },
					})
				}
				renderBT := func() {
					SwitchCard(c.Key("bt-card"), "Bluetooth", &bt, SwitchCardOptions{
						LeadingIcon: mdicon.Add.Draw,
					})
				}
				if swapped {
					renderBT()
					renderWifi()
				} else {
					renderWifi()
					renderBT()
				}
			})
		})
	}, 380, 220)

	swInfo, ok := tt.AccessByRole(ui.RoleSwitch, "Wi-Fi")
	if !ok {
		t.Fatal("SwitchCard missing RoleSwitch named Wi-Fi")
	}
	if swInfo.Checked {
		t.Fatal("Wi-Fi switch should start unchecked")
	}

	// 1. Click directly on the Switch track -> toggles ONCE (false -> true).
	tt.ClickAt(center(swInfo.Bounds))
	if !wifi || wifiChanges != 1 {
		t.Fatalf("click on embedded Switch gave wifi=%v changes=%d, want true/1", wifi, wifiChanges)
	}

	// 2. Click on the left title area of the SwitchCard -> toggles ONCE (true -> false).
	tt.ClickAt(60, swInfo.Bounds.Y+swInfo.Bounds.H/2)
	if wifi || wifiChanges != 2 {
		t.Fatalf("click on card body gave wifi=%v changes=%d, want false/2", wifi, wifiChanges)
	}

	// 3. Keyboard Space while focused toggles ONCE (false -> true).
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if !wifi || wifiChanges != 3 {
		t.Fatalf("Space key gave wifi=%v changes=%d, want true/3", wifi, wifiChanges)
	}

	// 4. Reorder cards across frames: key isolation preserves state cleanly.
	swapped = true
	tt.Frame()
	swInfo, ok = tt.AccessByRole(ui.RoleSwitch, "Wi-Fi")
	if !ok || !swInfo.Checked {
		t.Fatalf("after reorder, Wi-Fi access=%+v ok=%v", swInfo, ok)
	}
	btInfo, ok := tt.AccessByRole(ui.RoleSwitch, "Bluetooth")
	if !ok || !btInfo.Checked {
		t.Fatalf("after reorder, Bluetooth access=%+v ok=%v", btInfo, ok)
	}

	tt.Move(2, 2)
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("SwitchCard did not settle")
	}
}

func TestSwitchCardDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	on := false
	changes := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 120).Background(canvasRGB).Padding(16).Children(func() {
				SwitchCard(c.Key("dis"), "Airplane Mode", &on, SwitchCardOptions{
					Disabled:  true,
					OnChanged: func(bool) { changes++ },
				})
			})
		})
	}, 360, 120)

	sw, ok := tt.AccessByRole(ui.RoleSwitch, "Airplane Mode")
	if !ok || !sw.Disabled {
		t.Fatalf("disabled SwitchCard access=%+v ok=%v", sw, ok)
	}
	tt.ClickAt(center(sw.Bounds))
	tt.ClickAt(60, sw.Bounds.Y+sw.Bounds.H/2)
	if on || changes != 0 {
		t.Fatalf("disabled SwitchCard toggled: on=%v changes=%d", on, changes)
	}
}

func TestActionTileInteractionMetaRTLAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	clicks := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(380, 180).Background(canvasRGB).Padding(16).Gap(10).Children(func() {
				ActionTile(c.Key("proxy"), "Routing Rules", ActionTileOptions{
					Subtitle:    "Direct, Proxy, Global",
					LeadingIcon: mdicon.Search.Draw,
					Meta:        "24 ms",
					Badge:       "3",
					Hairline:    true,
					OnClick:     func() { clicks++ },
				})
				ActionTile(c.Key("dis"), "Disabled Tile", ActionTileOptions{
					Disabled: true,
					OnClick:  func() { clicks += 100 },
				})
			})
		})
	}, 380, 180)

	info, ok := tt.AccessByRole(ui.RoleButton, "Routing Rules")
	if !ok {
		t.Fatal("ActionTile missing RoleButton")
	}
	if info.Description != "24 ms" {
		t.Fatalf("ActionTile Meta Description = %q, want \"24 ms\"", info.Description)
	}

	tt.ClickAt(center(info.Bounds))
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if clicks != 3 {
		t.Fatalf("ActionTile clicks=%d want 3", clicks)
	}

	dis, ok := tt.AccessByRole(ui.RoleButton, "Disabled Tile")
	if !ok || !dis.Disabled {
		t.Fatalf("Disabled ActionTile access=%+v ok=%v", dis, ok)
	}
	tt.ClickAt(center(dis.Bounds))
	if clicks != 3 {
		t.Fatalf("disabled ActionTile fired OnClick: clicks=%d", clicks)
	}

	tt.Move(2, 2)
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ActionTile animations did not settle")
	}

	// RTL sanity check.
	rtlClicks := 0
	rtlTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Direction(ui.RightToLeft).Size(360, 100).Background(canvasRGB).Padding(12).Children(func() {
				ActionTile(c.Key("rtl"), "Settings", ActionTileOptions{
					LeadingIcon: mdicon.Star.Draw,
					OnClick:     func() { rtlClicks++ },
				})
			})
		})
	}, 360, 100)
	rtlInfo, ok := rtlTT.AccessByRole(ui.RoleButton, "Settings")
	if !ok {
		t.Fatal("RTL ActionTile missing")
	}
	rtlTT.ClickAt(center(rtlInfo.Bounds))
	if rtlClicks != 1 {
		t.Fatalf("RTL ActionTile clicks=%d want 1", rtlClicks)
	}
}
