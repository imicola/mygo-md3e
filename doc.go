// Package md3e is a pure-Go Material Design 3 Expressive component library for
// MyGo's native GPU UI toolkit (github.com/egoist/mygo/ui). It draws every
// component itself, with no HTML, WebView or cgo, and runs on desktops
// (macOS, Windows, Linux) and in browsers (GOOS=js GOARCH=wasm).
//
// The library is layered:
//
//   - [github.com/imicola/mygo-md3e/tokens] holds the MD3E design tokens: HCT color science,
//     tonal palettes, dynamic color schemes, typography, shape and elevation.
//   - [github.com/imicola/mygo-md3e/motion] maps the MD3E spring specifications onto MyGo's
//     physics, converting damping ratios to physical damping coefficients.
//   - [github.com/imicola/mygo-md3e/shape] builds the 35 Material Expressive shapes and morphs
//     between them as closed cubic Bézier loops.
//   - [github.com/imicola/mygo-md3e/symbols] provides 252 zero-bloat rounded 24x24 vector icons
//     whose (*Symbol).Draw method values plug directly into [IconDraw].
//   - package md3e itself holds the components, from buttons and selection
//     controls to navigation and overlays.
//
// Components follow MyGo's value API: an element and its context live for one
// build pass only, state is held by the caller through pointers, and a
// component's key is given to the context before the component is built.
package md3e

// Version is the library version. It is a placeholder until the first release.
const Version = "0.0.0-dev"
