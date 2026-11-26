package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

func (a *appState) docsView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	class := md3e.ContentSizeClassOf(c)
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	// ReduceMotion immediately settles any active stream reveal without leaving pending frames.
	if a.docs.streamFeeding && reduce {
		a.docs.completeStream()
	}

	showDetail := a.docs.compactPane != docsPaneTree
	md3e.ListDetailPane(c.Key("docs-list-detail"), func() {
		a.docsTreePane(c, cs, class)
	}, func() {
		showOutline := a.docsOutlineVisible(class)
		md3e.SupportingPane(c.Key("docs-supporting"), func() {
			a.docsReaderPane(c, th, cs, class, reduce)
		}, func() {
			a.docsOutlinePane(c, th, cs, class, reduce)
		}, md3e.SupportingPaneOptions{
			SplitPercent:       &a.docs.readerPct,
			DefaultMainPercent: 72,
			ShowSupporting:     &showOutline,
			Label:              "Document reader and outline",
		})
	}, md3e.ListDetailPaneOptions{
		SplitWidth:       &a.docs.treeWidth,
		DefaultListWidth: 232,
		ShowDetail:       &showDetail,
		Label:            "Knowledge workspace",
	})
}

func (a *appState) docsOutlineVisible(class md3e.WindowSizeClass) bool {
	switch class {
	case md3e.WindowCompact:
		return a.docs.compactPane == docsPaneOutline
	case md3e.WindowMedium:
		return a.docs.mediumOutline
	default:
		return a.docs.showOutline
	}
}

func (a *appState) setDocsOutlineVisible(class md3e.WindowSizeClass, show bool) {
	switch class {
	case md3e.WindowCompact:
		if show {
			a.docs.compactPane = docsPaneOutline
		} else if a.docs.compactPane == docsPaneOutline {
			a.docs.compactPane = docsPaneDoc
		}
	case md3e.WindowMedium:
		a.docs.mediumOutline = show
	default:
		a.docs.showOutline = show
	}
}

func (a *appState) docsTreePane(c *ui.Context, cs tokens.ColorScheme, class md3e.WindowSizeClass) {
	nodes, matched := a.docs.buildTreeNodes(func() {
		if class == md3e.WindowCompact {
			a.docs.compactPane = docsPaneDoc
		}
		a.dirty = true
	})

	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(cs.SurfaceContainerLow).Padding(12).Gap(10).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(6).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).MinWidth(0).Children(func() {
				if class == md3e.WindowCompact {
					if md3e.IconButton(c.Key("docs-tree-back"), "Back to document", symbols.ArrowBack.Draw, md3e.IconButtonOptions{
						Variant: md3e.IconButtonStandard,
						Size:    tokens.SizeXS,
					}).Clicked() {
						a.docs.compactPane = docsPaneDoc
						a.dirty = true
					}
				}
				md3e.IconBadge(c, symbols.FolderOpen.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgePrimaryTonal,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie6Sided(),
					Label:   "Knowledge base",
				})
				md3e.Text(c, "Knowledge Base", tokens.TitleSmall, true)
			})

			ui.Row(c).AlignItems(ui.Center).Gap(2).Shrink(0).Children(func() {
				if md3e.IconButton(c.Key("docs-tree-expand-all"), "Expand all folders", symbols.UnfoldMore.Draw, md3e.IconButtonOptions{
					Variant: md3e.IconButtonStandard,
					Size:    tokens.SizeXS,
				}).Clicked() {
					a.docs.setAllFoldersOpen(true)
					a.dirty = true
				}
				if md3e.IconButton(c.Key("docs-tree-collapse-all"), "Collapse all folders", symbols.UnfoldLess.Draw, md3e.IconButtonOptions{
					Variant: md3e.IconButtonStandard,
					Size:    tokens.SizeXS,
				}).Clicked() {
					a.docs.setAllFoldersOpen(false)
					a.dirty = true
				}
			})
		})

		md3e.TextField(c.Key("docs-tree-search"), &a.docs.treeFilter, md3e.TextFieldOptions{
			Variant:       md3e.TextFieldOutlined,
			Label:         "Filter docs",
			Placeholder:   "Heading, file, topic...",
			LeadingIcon:   symbols.Search.Draw,
			TrailingIcon:  symbols.Close.Draw,
			TrailingLabel: "Clear docs filter",
			OnTrailing: func() {
				a.docs.treeFilter = ""
				a.dirty = true
			},
			OnChanged: func(string) {
				a.dirty = true
			},
		})

		ui.Scroll(c.Key("docs-tree-scroll")).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().PaddingEnd(8).MinWidth(0).Children(func() {
				if len(nodes) == 0 {
					md3e.EmptyState(c, "No matching documents", md3e.EmptyStateOptions{
						Icon:               symbols.Description.Draw,
						IconShape:          shape.Cookie7Sided(),
						Description:        fmt.Sprintf("No notes match %q.", a.docs.treeFilter),
						PrimaryActionLabel: "Clear filter",
						OnPrimaryAction: func() {
							a.docs.treeFilter = ""
							a.dirty = true
						},
						Label: "No matching documents",
					})
					return
				}
				md3e.TreeView(c.Key("docs-tree"), nodes, md3e.TreeViewOptions{
					Selected:         &a.docs.selectedTreeKey,
					ShowIndentGuides: true,
					Label:            "Knowledge base directory",
					OnSelect: func(any) {
						a.dirty = true
					},
				})
			})
		})

		md3e.Divider(c)
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
			md3e.TextWith(c, fmt.Sprintf("%d/%d notes", matched, len(a.docs.docs)), md3e.TextOptions{
				Role:        tokens.LabelMedium,
				Mono:        true,
				TabularNums: true,
				Color:       cs.OnSurfaceVariant,
			})
			md3e.Tag(c, densityNames[a.densityIdx], md3e.TagOptions{
				Tone:    md3e.ToneSecondary,
				Variant: md3e.TagOutlined,
			})
		})
	})
}

func (a *appState) docsReaderPane(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass, reduce bool) {
	doc := a.docs.activeDoc()
	if doc == nil {
		return
	}

	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(cs.Surface).Children(func() {
		a.docsReaderChrome(c, th, cs, class, reduce, doc)

		if a.docs.streamFeeding {
			progOpts := md3e.ProgressOptions{
				Label:     "Markdown stream progress",
				Thickness: 4,
			}
			prog := a.docs.streamProgress()
			if reduce {
				md3e.LinearProgress(c.Key("docs-stream-linear"), prog, progOpts)
			} else {
				md3e.LinearWavyProgress(c.Key("docs-stream-wavy"), prog, progOpts)
			}
		}

		ui.Scroll(c.Key("docs-scroll")).TrackScroll(&a.docs.docScroll).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				ui.Column(c).FillWidth().MaxWidth(780).Padding(18, 24, 44, 24).Gap(14).Children(func() {
					a.docsMetaHeader(c, cs, doc)

					displayed := a.docs.renderedMarkdown()
					if a.docs.showRawSource {
						docview.CodeBlock(c.Key("docs-raw-source"), displayed, docview.CodeBlockOptions{
							Language:        "markdown",
							Title:           doc.fileName,
							ShowLineNumbers: true,
							Wrap:            true,
							Disabled:        a.docs.streamFeeding,
							Snackbar:        a.snack,
							OnCopy: func(string) {
								a.dirty = true
							},
						})
					} else {
						mdOpts := docview.MarkdownOptions{
							State:        &a.docs.docState,
							Scroll:       &a.docs.docScroll,
							HeadingRules: true,
							Snackbar:     a.snack,
							OnCopyCode: func(_ string, lang string) {
								if lang == "" {
									lang = "snippet"
								}
								a.dirty = true
							},
							OnLink: func(url string) {
								a.handleDocLink(c, th, reduce, url)
							},
						}
						if !a.docs.streamFeeding {
							mdOpts.OnTask = func(line int, checked bool) {
								if a.docs.toggleActiveTask(line, checked) {
									a.dirty = true
									stateWord := "checked"
									if !checked {
										stateWord = "unchecked"
									}
									a.snack.Show(md3e.SnackbarOptions{
										Message: fmt.Sprintf("Task line %d %s", line, stateWord),
									})
								}
							}
						}
						if a.docs.streamFeeding {
							docview.MarkdownStream(c.Key("docs-markdown"), displayed, mdOpts)
						} else {
							docview.Markdown(c.Key("docs-markdown"), displayed, mdOpts)
						}
					}

					a.docsFooterPager(c, cs)
				})
			})
		})
	})
}

func (a *appState) docsReaderChrome(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass, reduce bool, doc *docEntry) {
	crumbs := make([]md3e.BreadcrumbItem, 0, len(doc.folderPath)+2)
	crumbs = append(crumbs, md3e.BreadcrumbItem{
		Label: "Docs",
		Icon:  symbols.Home.Draw,
		OnClick: func() {
			a.docs.treeFilter = ""
			a.docs.setAllFoldersOpen(true)
			if class == md3e.WindowCompact {
				a.docs.compactPane = docsPaneTree
			}
			a.dirty = true
		},
	})
	for _, fid := range doc.folderPath {
		folder := a.docs.findFolder(fid)
		if folder == nil {
			continue
		}
		fRef := folder
		crumbs = append(crumbs, md3e.BreadcrumbItem{
			Label: fRef.title,
			Icon:  symbols.Folder.Draw,
			OnClick: func() {
				fRef.open = true
				fRef.filterOpen = true
				a.docs.selectedTreeKey = "folder:" + fRef.id
				if class == md3e.WindowCompact {
					a.docs.compactPane = docsPaneTree
				}
				a.dirty = true
			},
		})
	}
	crumbs = append(crumbs, md3e.BreadcrumbItem{
		Label: doc.fileName,
		Icon:  doc.icon,
	})

	statusText, statusTone, pulsing := "Ready", md3e.ToneNeutral, false
	switch {
	case a.docs.streamFeeding:
		statusText, statusTone, pulsing = "Streaming", md3e.TonePrimary, !reduce
	case doc.edited:
		statusText, statusTone = "Edited", md3e.ToneTertiary
	}

	ui.Column(c).
		FillWidth().
		Shrink(0).
		Padding(10, 16, 10, 16).
		Gap(8).
		Background(cs.SurfaceContainerLowest).
		BorderWidth(0, 0, 1, 0).
		BorderColor(cs.OutlineVariant.Alpha(0.5)).
		Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					md3e.Breadcrumb(c.Key("docs-breadcrumb"), crumbs, md3e.BreadcrumbOptions{
						Label: "Document path",
					})
				})
				md3e.StatusPill(c, statusText, md3e.StatusPillOptions{
					Tone:    statusTone,
					Pulsing: pulsing,
				})
			})

			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
					md3e.MorphIconButton(c.Key("docs-stream-toggle"), &a.docs.streamFeeding, "Stream document",
						symbols.PlayArrow.Draw, md3e.MorphIconButtonOptions{
							Variant:       md3e.ShapeButtonTonal,
							Size:          tokens.SizeXS,
							Shape:         shape.Cookie6Sided(),
							SelectedShape: shape.Square(),
							SelectedIcon:  symbols.Stop.Draw,
							OnChanged: func(on bool) {
								if on {
									a.docs.startStream(reduce)
									if reduce {
										a.snack.Show(md3e.SnackbarOptions{
											Message: "Revealed immediately (Reduce motion)",
										})
									}
								} else {
									a.docs.stopStream()
								}
								a.dirty = true
							},
						})
					md3e.Text(c, streamToggleHint(a.docs.streamFeeding), tokens.LabelMedium).
						TextColor(cs.OnSurfaceVariant)
					if a.docs.streamFeeding {
						if md3e.Button(c.Key("docs-stream-complete"), "Complete", md3e.ButtonOptions{
							Variant:     md3e.ButtonTonal,
							Size:        tokens.SizeXS,
							LeadingIcon: symbols.DoneAll.Draw,
						}).Clicked() {
							a.docs.completeStream()
							a.dirty = true
						}
					}
					if doc.edited {
						if md3e.Button(c.Key("docs-reset-tasks"), "Reset tasks", md3e.ButtonOptions{
							Variant: md3e.ButtonText,
							Size:    tokens.SizeXS,
						}).Clicked() {
							a.docs.resetActiveDoc()
							a.dirty = true
						}
					}
				})

				ui.Row(c).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
					if class == md3e.WindowCompact {
						treeShown := a.docs.compactPane == docsPaneTree
						md3e.FilterChip(c.Key("docs-files-chip"), &treeShown, "Files", md3e.FilterChipOptions{
							LeadingIcon: symbols.Folder.Draw,
							OnChanged: func(bool) {
								a.docs.compactPane = docsPaneTree
								a.dirty = true
							},
						})
					}
					md3e.FilterChip(c.Key("docs-source-chip"), &a.docs.showRawSource, "Source", md3e.FilterChipOptions{
						LeadingIcon: symbols.Code.Draw,
						OnChanged: func(bool) {
							a.dirty = true
						},
					})
					outlineShown := a.docsOutlineVisible(class)
					headingCount := len(a.docs.docState.Items())
					outlineLabel := fmt.Sprintf("Outline (%d)", headingCount)
					md3e.FilterChip(c.Key("docs-outline-chip"), &outlineShown, outlineLabel, md3e.FilterChipOptions{
						LeadingIcon: symbols.FormatListBulleted.Draw,
						OnChanged: func(next bool) {
							a.setDocsOutlineVisible(class, next)
							a.dirty = true
						},
					})
				})
			})
		})
}

func streamToggleHint(feeding bool) string {
	if feeding {
		return "Streaming..."
	}
	return "Stream doc"
}

func (a *appState) docsMetaHeader(c *ui.Context, cs tokens.ColorScheme, doc *docEntry) {
	items := a.docs.docState.Items()
	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerLow,
		ShapeScale: tokens.ShapeLarge,
		Hairline:   true,
	}).FillWidth().Padding(12, 16).Row().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Wrap().Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				md3e.Tag(c, doc.category, md3e.TagOptions{
					Tone:    md3e.TonePrimary,
					Variant: md3e.TagTonal,
				})
				md3e.TextWith(c, fmt.Sprintf("%s read · %d sections", doc.readTime, len(items)), md3e.TextOptions{
					Role:        tokens.LabelMedium,
					Mono:        true,
					TabularNums: true,
					Color:       cs.OnSurfaceVariant,
				})
			})
			md3e.TextWith(c, doc.summary, md3e.TextOptions{
				Role:  tokens.BodySmall,
				Color: cs.OnSurfaceVariant,
			})
		})
	})
}

func (a *appState) docsFooterPager(c *ui.Context, cs tokens.ColorScheme) {
	idx := a.docs.activeDocIndex()
	var prevDoc, nextDoc *docEntry
	if idx > 0 {
		prevDoc = &a.docs.docs[idx-1]
	}
	if idx+1 < len(a.docs.docs) {
		nextDoc = &a.docs.docs[idx+1]
	}
	rtl := c.Direction() == ui.RightToLeft
	prevIcon := symbols.ArrowBack.Draw
	nextIcon := symbols.ArrowForward.Draw
	if rtl {
		prevIcon = symbols.ArrowForward.Draw
		nextIcon = symbols.ArrowBack.Draw
	}

	md3e.Divider(c)
	ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Wrap().Children(func() {
		if prevDoc != nil {
			if md3e.Button(c.Key("docs-prev-doc"), "Prev: "+prevDoc.fileName, md3e.ButtonOptions{
				Variant:     md3e.ButtonOutlined,
				Size:        tokens.SizeS,
				LeadingIcon: prevIcon,
			}).Clicked() {
				a.docs.stepDoc(-1)
				a.dirty = true
			}
		} else {
			ui.Box(c)
		}

		if nextDoc != nil {
			if md3e.Button(c.Key("docs-next-doc"), "Next: "+nextDoc.fileName, md3e.ButtonOptions{
				Variant:      md3e.ButtonTonal,
				Size:         tokens.SizeS,
				TrailingIcon: nextIcon,
			}).Clicked() {
				a.docs.stepDoc(1)
				a.dirty = true
			}
		}
	})
}

func (a *appState) docsOutlinePane(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass, reduce bool) {
	items := a.docs.docState.Items()
	maxDepth := 0
	if a.docs.outlineDepthIdx == 1 {
		maxDepth = 2
	}

	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(cs.SurfaceContainerLow).Padding(12).Gap(10).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(6).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).MinWidth(0).Children(func() {
				if class == md3e.WindowCompact {
					if md3e.IconButton(c.Key("docs-outline-back"), "Back to document", symbols.ArrowBack.Draw, md3e.IconButtonOptions{
						Variant: md3e.IconButtonStandard,
						Size:    tokens.SizeXS,
					}).Clicked() {
						a.docs.compactPane = docsPaneDoc
						a.dirty = true
					}
				}
				md3e.IconBadge(c, symbols.FormatListBulleted.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeTertiaryTonal,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie4Sided(),
					Label:   "On this page",
				})
				md3e.Text(c, "On this page", tokens.TitleSmall, true)
			})
			md3e.Tag(c, fmt.Sprintf("%d", len(items)), md3e.TagOptions{
				Tone:    md3e.ToneTertiary,
				Variant: md3e.TagOutlined,
			})
		})

		md3e.SegmentedTabs(c.Key("docs-outline-depth"), []md3e.TabItem{
			{Label: "All"},
			{Label: "H1–H2"},
		}, md3e.PillTabsOptions{
			Selected:  &a.docs.outlineDepthIdx,
			FullWidth: true,
			Label:     "Outline heading depth",
			Changed:   func(int) { a.dirty = true },
		})

		ui.Scroll(c.Key("docs-outline-scroll")).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().PaddingEnd(8).MinWidth(0).Children(func() {
				docview.Outline(c.Key("docs-outline"), &a.docs.docState, docview.OutlineOptions{
					Scroll:    &a.docs.docScroll,
					MaxDepth:  maxDepth,
					EmptyText: "No headings in this document.",
					Label:     "Document outline",
					OnSelect: func(docview.OutlineItem) {
						if class == md3e.WindowCompact {
							a.docs.compactPane = docsPaneDoc
						}
						a.dirty = true
					},
				})
			})
		})

		md3e.Divider(c)
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
			activeSlug := a.docs.docState.ActiveSlug()
			slugText := "Top of note"
			if activeSlug != "" {
				slugText = "#" + activeSlug
			}
			md3e.TextWith(c, slugText, md3e.TextOptions{
				Role:     tokens.LabelSmall,
				Mono:     true,
				MaxLines: 1,
				Truncate: md3e.TruncateEnd,
				Color:    cs.OnSurfaceVariant,
			}).Grow(1).MinWidth(0)

			if md3e.Button(c.Key("docs-scroll-top"), "Top", md3e.ButtonOptions{
				Variant:     md3e.ButtonText,
				Size:        tokens.SizeXS,
				LeadingIcon: symbols.ArrowUpward.Draw,
			}).Clicked() {
				anim := th.Animation(motion.DefaultSpatial)
				if reduce {
					anim = ui.Animation{}
				}
				a.docs.docScroll.ScrollTo(0, 0, anim)
				a.dirty = true
			}
		})
	})
}

func (a *appState) handleDocLink(c *ui.Context, th *md3e.Theme, reduce bool, url string) {
	if docID, ok := strings.CutPrefix(url, "doc://"); ok {
		if a.docs.selectDoc(docID) {
			a.dirty = true
			if doc := a.docs.activeDoc(); doc != nil {
				a.snack.Show(md3e.SnackbarOptions{Message: "Opened " + doc.fileName})
			}
			return
		}
	}
	if strings.HasPrefix(url, "#") {
		a.docs.docState.JumpTo(url)
		a.dirty = true
		return
	}
	c.WriteClipboard(url)
	a.snack.Show(md3e.SnackbarOptions{Message: "Copied link: " + url})
	_ = th
	_ = reduce
}
