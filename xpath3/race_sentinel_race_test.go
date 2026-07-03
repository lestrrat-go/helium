//go:build race

package xpath3_test

// raceEnabled reports whether the binary was built with the race detector.
// Race instrumentation inflates allocation counts, so allocation-bound
// assertions are relaxed when it is active. There is no runtime API to detect
// the race detector, so it is derived from the build tag.
const raceEnabled = true
