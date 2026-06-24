package heliumcmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSplitSearchPath verifies the --path (DTD/entity search path) splitter
// keeps a Windows drive-letter prefix attached to its directory while still
// splitting a genuine colon-separated list. A naive strings.Split(':') would
// shatter "D:\\dtd" into "D" and "\\dtd", so a DTD resolved via --path becomes
// unfindable on Windows and validation spuriously fails (TestLintOutput
// OverLaterReadDTDSucceeds, exit 3). The shapes are plain strings, so the
// Windows behavior is exercised on Linux.
func TestSplitSearchPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want []string
	}{
		{`D:\a\helium\dtd`, []string{`D:\a\helium\dtd`}},
		{"C:/x/dtd", []string{"C:/x/dtd"}},
		{`D:\a:E:\b`, []string{`D:\a`, `E:\b`}}, // two drive paths, colon-joined
		{"/a/b:/c/d", []string{"/a/b", "/c/d"}}, // POSIX list still splits
		{"single", []string{"single"}},
		{"", []string{""}},
		{"D:", []string{"D:"}},                // bare drive
		{"a:b", []string{"a", "b"}},           // single-letter but no separator after -> list
		{"/x:D:\\y", []string{"/x", "D:\\y"}}, // mixed POSIX + drive
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, splitSearchPath(tc.in), "in=%q", tc.in)
	}
}
