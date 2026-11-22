// Package charts provides Material Design 3 Expressive (MD3E) real-time and
// statistical data visualization components built over MyGo's headless
// github.com/egoist/mygo/ui/chart primitives.
//
// The package is strictly isolated from the root md3e package so applications
// that do not render charts never link charts or ui/chart. It has no package
// init() registries and no global mutable tables, preserving per-component
// dead-code elimination.
//
// Components follow the same conventions as package md3e:
//   - The first parameter is c *ui.Context.
//   - Call c.Key(...) before constructing stateful or animated charts.
//   - Optional configuration is passed via a single trailing Options struct.
//   - Custom icons accept md3e.IconDraw callbacks so callers remain decoupled
//     from internal icon packages.
package charts
