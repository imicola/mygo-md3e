// Package docview provides Material Design 3 Expressive (MD3E) document,
// Markdown, callout, code card, and scroll-linked outline components built over
// MyGo's github.com/egoist/mygo/ui/markdown engine.
//
// The package is strictly isolated from the root md3e package so applications
// that do not display Markdown or documentation never link docview or
// ui/markdown. It has no package init() functions or global mutable tables,
// preserving dead-code elimination.
package docview
