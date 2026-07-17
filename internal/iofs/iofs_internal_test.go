package iofs

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// A ConfinedDir carrying a construction error (filepath.Abs failed, e.g. the
// working directory was removed so os.Getwd fails) fails closed: Open returns
// that error rather than resolving a relative root against whatever working
// directory is current at Open time. The error field is injected directly here
// because filepath.Abs failing deterministically would require racing the
// process working directory.
func TestConfinedDirFailsClosedOnConstructionError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("getwd failed")
	c := ConfinedDir{root: "relative/root", err: sentinel}

	_, err := c.Open("sub.dtd")
	require.ErrorIs(t, err, sentinel,
		"a ConfinedDir with a construction error must return it from Open")
}

func TestIsInvalidNameOpenError(t *testing.T) {
	t.Parallel()

	require.True(t, isInvalidNameOpenError("linux", fs.ErrInvalid))
	require.True(t, isInvalidNameOpenError(goosWindows, &fs.PathError{
		Op:   "open",
		Path: "http://example.com/missing.xsd",
		Err:  syscall.Errno(windowsInvalidNameCode),
	}))
	require.False(t, isInvalidNameOpenError("linux", syscall.Errno(123)))
	require.False(t, isInvalidNameOpenError(goosWindows, errors.New("HTTP 500")))
}
