package iofs

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

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
