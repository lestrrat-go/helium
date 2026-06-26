package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainedPath(t *testing.T) {
	root := filepath.Join("/tmp", "qt3ts", "source")

	t.Run("normal relative path", func(t *testing.T) {
		got, err := containedPath(root, "fn/abs.xml")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, "fn", "abs.xml"), got)
	})

	t.Run("empty path rejected", func(t *testing.T) {
		_, err := containedPath(root, "")
		require.Error(t, err)
	})

	t.Run("posix absolute path rejected", func(t *testing.T) {
		_, err := containedPath(root, "/etc/passwd")
		require.Error(t, err)
	})

	t.Run("windows absolute path rejected", func(t *testing.T) {
		_, err := containedPath(root, `C:\Windows\system32`)
		require.Error(t, err)
	})

	t.Run("backslash-rooted path rejected", func(t *testing.T) {
		_, err := containedPath(root, `\rooted\escape`)
		require.Error(t, err)
	})

	t.Run("dot-dot escape rejected", func(t *testing.T) {
		_, err := containedPath(root, "../../xpath3/pwn.go")
		require.Error(t, err)
	})

	t.Run("interior dot-dot that stays inside is allowed", func(t *testing.T) {
		got, err := containedPath(root, "fn/../math/abs.xml")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, "math", "abs.xml"), got)
	})

	t.Run("interior dot-dot that escapes is rejected", func(t *testing.T) {
		_, err := containedPath(root, "fn/../../escape.xml")
		require.Error(t, err)
	})
}
