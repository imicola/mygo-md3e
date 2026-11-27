package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/charts"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

var connColumns = []md3e.DataTableColumn{
	{Title: "Host", Width: 0, MinWidth: 160, Sortable: true, Mono: true},
	{Title: "Process", Width: 118, Sortable: true, Mono: true},
	{Title: "Rule", Width: 170, Sortable: true},
	{Title: "Route", Width: 84, Sortable: true},
	{Title: "Proto", Width: 68},
	{Title: "Latency", Width: 88, Numeric: true, Sortable: true},
	{Title: "Down", Width: 96, Numeric: true, Sortable: true},
	{Title: "Up", Width: 88, Numeric: true, Sortable: true},
}

var dnsColumns = []md3e.DataTableColumn{
	{Title: "Domain", Width: 0, MinWidth: 180, Sortable: true, Mono: true},
	{Title: "Type", Width: 76, Sortable: true, Mono: true},
	{Title: "Resolver", Width: 196, Sortable: true, Mono: true},
	{Title: "Status", Width: 92, Sortable: true},
	{Title: "TTL", Width: 76, Numeric: true, Sortable: true},
	{Title: "Latency", Width: 88, Numeric: true, Sortable: true},
}

var routingModeNames = []string{"Rule", "Global", "Direct"}

func (a *appState) monitorView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	a.mon.ensureFiltered()

	ui.Scroll(c.Key("monitor-scroll")).Fill().Children(func() {
		ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(1240).Padding(18, 20, 44, 20).Gap(14).Children(func() {
				a.monitorHeader(c, cs)
				a.monitorBentoGrid(c)
				a.monitorLedgerSection(c)
			})
		})
	})
}

func (a *appState) monitorHeader(c *ui.Context, cs tokens.ColorScheme) {
	th := md3e.CurrentTheme(c)
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	statusText, statusTone := "Paused", md3e.ToneNeutral
	if a.mon.running {
		statusText, statusTone = "Live · 2 Hz", md3e.ToneSuccess
	}

	ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Wrap().Children(func() {
		ui.Column(c).MinWidth(0).Gap(2).Children(func() {
			md3e.Text(c, "Traffic & Process Monitor", tokens.HeadlineSmall, true)
			md3e.Text(c, "Real-time throughput, Go runtime memory breakdown, and virtualized 10K–100K ledger", tokens.BodySmall).
				TextColor(cs.OnSurfaceVariant)
		})
		ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
			md3e.StatusPill(c, statusText, md3e.StatusPillOptions{
				Tone:    statusTone,
				Pulsing: a.mon.running && !reduce,
			})
			md3e.Tag(c, densityNames[a.densityIdx], md3e.TagOptions{
				Tone:        md3e.TonePrimary,
				Variant:     md3e.TagOutlined,
				LeadingIcon: symbols.ViewList.Draw,
			})
		})
	})
}

func (a *appState) monitorBentoGrid(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	md3e.BentoGrid(c, func() {
		// 1. Hero Engine Control Card (2x1 BentoSpan)
		a.heroEngineCard(c, th, cs, reduce)

		// 2. Live Download Sparkline Card (1x1)
		downToneCol := charts.SeriesColor(th, 0)
		charts.MetricCard(c.Key("metric-download"), charts.MetricCardOptions{
			Title:      "Download",
			Subtitle:   "Aggregated ingress",
			Icon:       symbols.ArrowDownward.Draw,
			IconTone:   md3e.TonePrimary,
			Badge:      "Live",
			BadgeTone:  md3e.TonePrimary,
			Pulsing:    a.mon.running && !reduce,
			Value:      a.mon.downSpeedText,
			Unit:       "MB/s",
			Secondary:  a.mon.downPeakText,
			ValueColor: downToneCol,
			Sparkline: &charts.AreaSparklineOptions{
				Ring:         a.mon.downRing,
				Color:        downToneCol,
				ShowEndDot:   true,
				BaselineZero: true,
				Curve:        charts.CurveSmooth,
				Label:        "Download throughput trend",
			},
		})

		// 3. Live Upload Sparkline Card (1x1)
		upToneCol := charts.SeriesColor(th, 1)
		charts.MetricCard(c.Key("metric-upload"), charts.MetricCardOptions{
			Title:      "Upload",
			Subtitle:   "Aggregated egress",
			Icon:       symbols.ArrowUpward.Draw,
			IconTone:   md3e.ToneTertiary,
			Badge:      "Uplink",
			BadgeTone:  md3e.ToneTertiary,
			Value:      a.mon.upSpeedText,
			Unit:       "MB/s",
			Secondary:  a.mon.upPeakText,
			ValueColor: upToneCol,
			Sparkline: &charts.AreaSparklineOptions{
				Ring:         a.mon.upRing,
				Color:        upToneCol,
				ShowEndDot:   true,
				BaselineZero: true,
				Curve:        charts.CurveSmooth,
				Label:        "Upload throughput trend",
			},
		})

		// 4. Live Process Memory Self-Inspection Card (2x1 BentoSpan)
		a.memoryFootprintCard(c, th, cs)

		// 5. DNS & Routing Health Card (2x1 BentoSpan)
		a.dnsHealthCard(c, cs, reduce)
	}, md3e.BentoGridOptions{Label: "Monitor overview grid"})
}

func (a *appState) heroEngineCard(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, reduce bool) {
	fromCol := cs.PrimaryContainer.MixOklab(cs.SurfaceContainerHigh, 0.20)
	if th.Dark {
		fromCol = cs.PrimaryContainer.MixOklab(cs.SurfaceContainerHigh, 0.36)
	}
	gradient := ui.LinearGradient{
		From:  fromCol,
		To:    cs.SurfaceContainerHigh,
		Angle: 126,
		Oklab: true,
	}
	orbA := float32(0.16)
	orbB := float32(0.11)
	if th.Dark {
		orbA = 0.22
		orbB = 0.14
	}
	backdrop := func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		s1 := [3]ui.GradientStop{
			{Color: cs.Primary.Alpha(orbA), At: 0},
			{Color: cs.Primary.Alpha(orbA * 0.35), At: 0.48},
			{Color: cs.Primary.Alpha(0), At: 1},
		}
		p.FillRadialGradient(r, ui.RadialGradient{
			Stops:  s1[:],
			Center: ui.Point{X: 0.86, Y: 0.16},
			Oklab:  true,
		}, 0)
		s2 := [3]ui.GradientStop{
			{Color: cs.Tertiary.Alpha(orbB), At: 0},
			{Color: cs.Tertiary.Alpha(orbB * 0.35), At: 0.45},
			{Color: cs.Tertiary.Alpha(0), At: 1},
		}
		p.FillRadialGradient(r, ui.RadialGradient{
			Stops:  s2[:],
			Center: ui.Point{X: 0.92, Y: 0.86},
			Oklab:  true,
		}, 0)
	}

	status, tone := "Engine Paused", md3e.ToneNeutral
	if a.mon.running {
		status, tone = "Proxy Active", md3e.ToneSuccess
	}

	card := md3e.Card(c, func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(12).Children(func() {
				md3e.IconBadge(c, symbols.Router.Draw, md3e.IconBadgeOptions{
					Variant:       md3e.IconBadgeFilled,
					ContainerSize: 42,
					IconSize:      22,
					Shape:         shape.Cookie9Sided(),
					Label:         "Tunnel core",
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					md3e.Text(c, "Tunnel Core", tokens.TitleMedium, true)
					tunLabel := "TUN off"
					if a.mon.tunEnabled {
						tunLabel = "TUN on"
					}
					fakeLabel := "System DNS"
					if a.mon.fakeIPEnabled {
						fakeLabel = "Fake-IP"
					}
					md3e.Text(c, routingModeNames[a.mon.routingMode]+" routing · "+tunLabel+" · "+fakeLabel, tokens.BodySmall).
						TextColor(cs.OnSurfaceVariant)
				})
				md3e.MorphIconButton(c.Key("engine-toggle"), &a.mon.running, "Engine toggle",
					symbols.PlayArrow.Draw, md3e.MorphIconButtonOptions{
						Variant:       md3e.ShapeButtonFilled,
						Size:          tokens.SizeM,
						Shape:         shape.Cookie9Sided(),
						SelectedShape: shape.Circle(),
						SelectedIcon:  symbols.Pause.Draw,
						OnChanged: func(on bool) {
							if !on {
								a.mon.lastTick = time.Time{}
							}
							a.dirty = true
						},
					})
			})

			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
				md3e.StatusPill(c, status, md3e.StatusPillOptions{
					Tone:    tone,
					Pulsing: a.mon.running && !reduce,
				})
				md3e.Tag(c, fmt.Sprintf("%d conns", a.mon.activeConns), md3e.TagOptions{
					Tone:    md3e.TonePrimary,
					Variant: md3e.TagOutlined,
				})
				md3e.Tag(c, a.mon.dnsQPSText, md3e.TagOptions{
					Tone:    md3e.ToneTertiary,
					Variant: md3e.TagOutlined,
				})
			})

			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				md3e.SegmentedTabs(c.Key("routing-mode"), []md3e.TabItem{
					{Label: routingModeNames[0]},
					{Label: routingModeNames[1]},
					{Label: routingModeNames[2]},
				}, md3e.PillTabsOptions{
					Selected: &a.mon.routingMode,
					Label:    "Routing mode",
					Changed:  func(int) { a.dirty = true },
				})
				md3e.FilterChip(c.Key("tun-chip"), &a.mon.tunEnabled, "TUN", md3e.FilterChipOptions{
					LeadingIcon: symbols.Shield.Draw,
				})
				md3e.FilterChip(c.Key("fakeip-chip"), &a.mon.fakeIPEnabled, "Fake-IP", md3e.FilterChipOptions{
					LeadingIcon: symbols.Dns.Draw,
				})
			})
		})
	}, md3e.CardOptions{
		Variant:        md3e.CardFilled,
		Padding:        16,
		ShapeScale:     tokens.ShapeExtraLarge,
		Hairline:       true,
		LinearGradient: &gradient,
		Backdrop:       backdrop,
	})
	md3e.BentoSpan(card, 2, 1)
}

func (a *appState) memoryFootprintCard(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme) {
	mem := a.mon.mem
	segments := []charts.StackedSegment{
		{
			Label:      "Live Heap",
			Value:      mem.heapAlloc,
			Color:      charts.SeriesColor(th, 0),
			ValueLabel: formatBytesIEC(mem.heapAlloc),
		},
		{
			Label:      "Heap Cache",
			Value:      mem.heapFree,
			Color:      charts.SeriesColor(th, 2),
			ValueLabel: formatBytesIEC(mem.heapFree),
		},
		{
			Label:      "Stack & Runtime",
			Value:      mem.stackSys,
			Color:      charts.SeriesColor(th, 1),
			ValueLabel: formatBytesIEC(mem.stackSys),
		},
	}

	card := charts.MetricCard(c.Key("metric-memory"), charts.MetricCardOptions{
		Title:      "Process Memory",
		Subtitle:   "Live runtime.MemStats self-check",
		Icon:       symbols.Memory.Draw,
		IconTone:   md3e.TonePrimary,
		Badge:      mem.gcText,
		BadgeTone:  md3e.ToneSecondary,
		Value:      mem.heapText,
		Unit:       "Go Heap",
		Secondary:  mem.sysText,
		ShapeScale: tokens.ShapeLargeIncreased,
		Footer: func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
				charts.StackedProgress(c.Key("mem-stacked"), segments, charts.StackedProgressOptions{
					Total:       mem.sysTotal,
					ShowLegend:  true,
					ShowPercent: true,
					Label:       "Process memory breakdown",
					ValueFormatter: func(seg charts.StackedSegment, pct float64) string {
						return fmt.Sprintf("%s (%.0f%%)", formatBytesIEC(seg.Value), pct)
					},
				})
				ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
					md3e.Text(c, "Pure Go UI · zero WebView / Chromium heap", tokens.BodySmall).
						TextColor(cs.OnSurfaceVariant)
					if md3e.Button(c.Key("run-gc"), "Run GC", md3e.ButtonOptions{
						Variant:     md3e.ButtonTonal,
						Size:        tokens.SizeXS,
						LeadingIcon: symbols.Refresh.Draw,
					}).Clicked() {
						a.mon.triggerGC()
						a.snack.Show(md3e.SnackbarOptions{
							Message: fmt.Sprintf("GC complete · Live heap %s", a.mon.mem.heapText),
						})
					}
				})
			})
		},
	})
	md3e.BentoSpan(card, 2, 1)
}

func (a *appState) dnsHealthCard(c *ui.Context, cs tokens.ColorScheme, reduce bool) {
	card := charts.MetricCard(c.Key("metric-dns"), charts.MetricCardOptions{
		Title:      "DNS Cache Hit Rate",
		Subtitle:   "DoH + Fake-IP resolver pool",
		Icon:       symbols.Dns.Draw,
		IconTone:   md3e.ToneTertiary,
		Badge:      a.mon.dnsQPSText,
		BadgeTone:  md3e.ToneSuccess,
		Value:      a.mon.dnsHitText,
		Unit:       "hit ratio",
		Secondary:  "3 upstream DoH nodes",
		ShapeScale: tokens.ShapeLargeIncreased,
		Footer: func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
				progOpts := md3e.ProgressOptions{
					Label:     "DNS cache hit ratio",
					Thickness: 6,
					Buffer:    min(1.0, a.mon.dnsHitRate+0.07),
				}
				if a.mon.running && !reduce {
					md3e.LinearWavyProgress(c.Key("dns-hit-wavy"), a.mon.dnsHitRate, progOpts)
				} else {
					md3e.LinearProgress(c.Key("dns-hit-linear"), a.mon.dnsHitRate, progOpts)
				}
				ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
					ui.Row(c).Gap(6).Wrap().Children(func() {
						md3e.Tag(c, "DoH", md3e.TagOptions{Tone: md3e.TonePrimary, Variant: md3e.TagOutlined})
						md3e.Tag(c, "EDNS", md3e.TagOptions{Tone: md3e.ToneSecondary, Variant: md3e.TagOutlined})
					})
					if md3e.Button(c.Key("reset-peaks"), "Reset peaks", md3e.ButtonOptions{
						Variant: md3e.ButtonText,
						Size:    tokens.SizeXS,
					}).Clicked() {
						a.mon.resetPeaks()
						a.snack.Show(md3e.SnackbarOptions{Message: "Throughput peaks reset"})
					}
				})
			})
		},
	})
	md3e.BentoSpan(card, 2, 1)
}

func (a *appState) monitorLedgerSection(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors

	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerLow,
		ShapeScale: tokens.ShapeLargeIncreased,
		Hairline:   true,
	}).FillWidth().Padding(16).Column().Gap(12).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Wrap().Children(func() {
			totalLabel := strconv.Itoa(a.mon.targetCount())
			connBadge := totalLabel
			dnsBadge := totalLabel
			if a.mon.tableMode == modeConnections {
				connBadge = strconv.Itoa(len(a.mon.filtered))
			} else {
				dnsBadge = strconv.Itoa(len(a.mon.filtered))
			}
			md3e.PillTabs(c.Key("ledger-mode"), []md3e.TabItem{
				{Label: "Connections", Icon: symbols.Public.Draw, Badge: connBadge},
				{Label: "DNS Queries", Icon: symbols.Dns.Draw, Badge: dnsBadge},
			}, md3e.PillTabsOptions{
				Selected: &a.mon.tableMode,
				Label:    "Ledger dataset",
				Changed: func(int) {
					a.mon.selectedRow = -1
					a.mon.selectedID = 0
					a.mon.markFilterDirty()
					a.dirty = true
				},
			})

			md3e.SegmentedTabs(c.Key("ledger-scale"), []md3e.TabItem{
				{Label: "10K rows"},
				{Label: "100K rows"},
			}, md3e.PillTabsOptions{
				Selected: &a.mon.scaleIdx,
				Label:    "Dataset scale",
				Changed: func(int) {
					a.mon.markFilterDirty()
					a.dirty = true
				},
			})
		})

		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(10).Wrap().Children(func() {
			ui.Box(c).Grow(1).MinWidth(220).Children(func() {
				md3e.TextField(c.Key("ledger-search"), &a.mon.query, md3e.TextFieldOptions{
					Variant:       md3e.TextFieldOutlined,
					Label:         "Filter host, process, or rule",
					Placeholder:   "github, go, DoH, Blocked...",
					LeadingIcon:   symbols.Search.Draw,
					TrailingIcon:  symbols.Close.Draw,
					TrailingLabel: "Clear table filter",
					OnTrailing: func() {
						a.mon.query = ""
						a.mon.markFilterDirty()
						a.dirty = true
					},
					OnChanged: func(string) {
						a.mon.markFilterDirty()
						a.dirty = true
					},
				})
			})

			ui.Row(c).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
				if a.mon.tableMode == modeDNS {
					md3e.FilterChip(c.Key("chip-dns-cached"), &a.mon.filterDNS[dnsCached], "Cached", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
					md3e.FilterChip(c.Key("chip-dns-doh"), &a.mon.filterDNS[dnsDoH], "DoH", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
					md3e.FilterChip(c.Key("chip-dns-blocked"), &a.mon.filterDNS[dnsBlocked], "Blocked", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
				} else {
					md3e.FilterChip(c.Key("chip-conn-proxy"), &a.mon.filterRoute[routeProxy], "Proxy", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
					md3e.FilterChip(c.Key("chip-conn-direct"), &a.mon.filterRoute[routeDirect], "Direct", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
					md3e.FilterChip(c.Key("chip-conn-blocked"), &a.mon.filterRoute[routeBlocked], "Blocked", md3e.FilterChipOptions{
						OnChanged: func(bool) { a.mon.markFilterDirty(); a.dirty = true },
					})
				}
				if a.mon.hasActiveFilters() {
					if md3e.Button(c.Key("clear-filters"), "Reset", md3e.ButtonOptions{
						Variant: md3e.ButtonText,
						Size:    tokens.SizeXS,
					}).Clicked() {
						a.mon.clearFilters()
						a.dirty = true
					}
				}
			})
		})

		a.mon.ensureFiltered()
		count := len(a.mon.filtered)
		sortPtr, sortByPtr := a.mon.activeSort()
		listState := a.mon.activeListState()

		tableKey := "conn-table"
		cols := connColumns
		tableLabel := "Connections virtual table"
		if a.mon.tableMode == modeDNS {
			tableKey = "dns-table"
			cols = dnsColumns
			tableLabel = "DNS queries virtual table"
		}

		srv := c.Services()
		md3e.DataTable(c.Key(tableKey), cols, count, func(row, col int) {
			a.renderLedgerCell(c, cs, row, col)
		}, md3e.DataTableOptions{
			State:     listState,
			Selected:  &a.mon.selectedRow,
			Sort:      sortPtr,
			SortBy:    sortByPtr,
			RoundRows: true,
			Hairline:  true,
			Label:     tableLabel,
			ItemKey: func(row int) any {
				return a.mon.recordIDAt(row)
			},
			RowLabel: func(row int) string {
				return a.mon.hostAt(row)
			},
			OnSortChanged: func(ui.SortOrder, []ui.SortOrder) {
				a.mon.markFilterDirty()
				a.dirty = true
				srv.Invalidate()
			},
			OnRowClick: func(row int) {
				a.mon.syncSelectedRow(row)
			},
			OnRowSubmit: func(row int) {
				a.mon.syncSelectedRow(row)
				if host := a.mon.hostAt(row); host != "" {
					srv.WriteClipboard(host)
					a.snack.Show(md3e.SnackbarOptions{Message: "Copied " + host})
				}
			},
			HeaderMenu: func(m *ui.Menu, col int) {
				a.buildLedgerHeaderMenu(m, cols, col)
			},
			CellMenu: func(m *ui.Menu, row, _ int) {
				a.buildLedgerCellMenu(srv, m, row)
			},
		}).FillWidth().Height(400)
		a.mon.syncSelectedRow(a.mon.selectedRow)

		if a.mon.selectedRow >= 0 && a.mon.selectedRow < len(a.mon.filtered) {
			a.selectedRowStrip(c, cs)
		}
	})
}

func (a *appState) renderLedgerCell(c *ui.Context, cs tokens.ColorScheme, row, col int) {
	if a.mon.tableMode == modeDNS {
		txt := a.mon.dnsCellText(row, col)
		opts := md3e.DataTableCellOptions{}
		switch col {
		case 0:
			opts.Mono = true
			opts.Truncate = md3e.TruncateMiddle
			opts.Emphasized = true
		case 1, 2:
			opts.Mono = true
			opts.Muted = true
			opts.Truncate = md3e.TruncateMiddle
		case 3:
			if row != a.mon.selectedRow {
				rec := &a.mon.dns[a.mon.filtered[row]]
				switch rec.status {
				case dnsCached:
					opts.Color = cs.Primary
				case dnsBlocked:
					opts.Color = cs.Error
				default:
					opts.Color = cs.Tertiary
				}
			}
		case 4, 5:
			opts.Numeric = true
		}
		md3e.DataTableCellText(c, txt, opts)
		return
	}

	txt := a.mon.connCellText(row, col)
	opts := md3e.DataTableCellOptions{}
	switch col {
	case 0:
		opts.Mono = true
		opts.Truncate = md3e.TruncateMiddle
		opts.Emphasized = true
	case 1:
		opts.Mono = true
	case 2, 4:
		opts.Muted = true
	case 3:
		if row != a.mon.selectedRow {
			rec := &a.mon.conns[a.mon.filtered[row]]
			switch rec.route {
			case routeProxy:
				opts.Color = cs.Primary
			case routeDirect:
				opts.Color = cs.Tertiary
			case routeBlocked:
				opts.Color = cs.Error
			}
		}
	case 5, 6, 7:
		opts.Numeric = true
	}
	md3e.DataTableCellText(c, txt, opts)
}

func (a *appState) buildLedgerHeaderMenu(m *ui.Menu, cols []md3e.DataTableColumn, col int) {
	var actions []md3e.DataTableMenuAction
	if col >= 0 && col < len(cols) && cols[col].Sortable {
		colTitle := cols[col].Title
		actions = append(actions,
			md3e.DataTableMenuAction{
				Label: "Sort ascending",
				OnClick: func() {
					s, sb := a.mon.activeSort()
					*s = ui.SortOrder{Column: colTitle}
					*sb = []ui.SortOrder{*s}
					a.mon.markFilterDirty()
					a.dirty = true
				},
			},
			md3e.DataTableMenuAction{
				Label: "Sort descending",
				OnClick: func() {
					s, sb := a.mon.activeSort()
					*s = ui.SortOrder{Column: colTitle, Descending: true}
					*sb = []ui.SortOrder{*s}
					a.mon.markFilterDirty()
					a.dirty = true
				},
			},
			md3e.DataTableMenuAction{Separator: true},
		)
	}
	actions = append(actions, md3e.DataTableMenuAction{
		Label: "Clear sort",
		OnClick: func() {
			s, sb := a.mon.activeSort()
			*s = ui.SortOrder{}
			*sb = nil
			a.mon.markFilterDirty()
			a.dirty = true
		},
	})
	md3e.ApplyDataTableMenuActions(m, actions)
}

func (a *appState) buildLedgerCellMenu(srv ui.Services, m *ui.Menu, row int) {
	host := a.mon.hostAt(row)
	if host == "" {
		return
	}
	md3e.ApplyDataTableMenuActions(m, []md3e.DataTableMenuAction{
		{
			Label: "Copy " + host,
			OnClick: func() {
				srv.WriteClipboard(host)
				a.snack.Show(md3e.SnackbarOptions{Message: "Copied " + host})
				srv.Invalidate()
			},
		},
		{
			Label: "Filter by host",
			OnClick: func() {
				a.mon.query = host
				a.mon.markFilterDirty()
				a.dirty = true
			},
		},
		{Separator: true},
		{
			Label: "Clear selection",
			OnClick: func() {
				a.mon.syncSelectedRow(-1)
				a.dirty = true
			},
		},
	})
}

func (a *appState) selectedRowStrip(c *ui.Context, cs tokens.ColorScheme) {
	row := a.mon.selectedRow
	host := a.mon.hostAt(row)
	var summary string
	if a.mon.tableMode == modeDNS {
		rec := &a.mon.dns[a.mon.filtered[row]]
		summary = fmt.Sprintf("%s · %s · %s · TTL %ds · %s",
			dnsTypeNames[rec.qtypeIdx], dnsResolvers[rec.resolverIdx],
			dnsStatusName[rec.status], rec.ttlSec, formatLatency10(rec.latency10))
	} else {
		rec := &a.mon.conns[a.mon.filtered[row]]
		summary = fmt.Sprintf("%s · %s:%d · %s · %s · ↓ %s / ↑ %s",
			processNames[rec.procIdx], protoNames[rec.proto], rec.port,
			routeNames[rec.route], formatLatency10(rec.latency10),
			formatTrafficKB(rec.downKB), formatTrafficKB(rec.upKB))
	}

	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerHigh,
		ShapeScale: tokens.ShapeMedium,
		Hairline:   true,
	}).FillWidth().Padding(10, 14).Row().AlignItems(ui.Center).Gap(10).Wrap().Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			md3e.TextWith(c, host, md3e.TextOptions{
				Role:       tokens.LabelLarge,
				Emphasized: true,
				Mono:       true,
				Truncate:   md3e.TruncateMiddle,
				MaxLines:   1,
				Color:      cs.OnSurface,
			})
			md3e.TextWith(c, summary, md3e.TextOptions{
				Role:        tokens.BodySmall,
				Mono:        true,
				TabularNums: true,
				Color:       cs.OnSurfaceVariant,
			})
		})
		if md3e.Button(c.Key("copy-selected-host"), "Copy host", md3e.ButtonOptions{
			Variant:     md3e.ButtonTonal,
			Size:        tokens.SizeXS,
			LeadingIcon: symbols.ContentCopy.Draw,
		}).Clicked() {
			c.WriteClipboard(host)
			a.snack.Show(md3e.SnackbarOptions{Message: "Copied " + host})
		}
		if md3e.Button(c.Key("clear-selected-row"), "Dismiss", md3e.ButtonOptions{
			Variant: md3e.ButtonText,
			Size:    tokens.SizeXS,
		}).Clicked() {
			a.mon.syncSelectedRow(-1)
		}
	})
}
