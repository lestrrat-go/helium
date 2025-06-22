package nsstack_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/stack"
	"github.com/lestrrat-go/helium/internal/stack/nsstack"
	"github.com/stretchr/testify/require"
)

func TestNsStack(t *testing.T) {
	s := nsstack.New()
	s.Push("xml", "http://www.w3.org/XML/1998/namespace")
	s.Push("ds", "http://www.w3.org/2000/09/xmldsig#")

	require.Equal(t, 2, s.Len(), "Len == 2")

	item := s.Lookup("ds")
	require.Equal(t, "http://www.w3.org/2000/09/xmldsig#", item, `Lookup("ds") succeeds`)

	item = s.Lookup("xml")
	require.NotEqual(t, stack.NilItem, item, `Lookup("xm") is not a NilItem`)

	require.Equal(t, "http://www.w3.org/XML/1998/namespace", item, `Lookup("xml") succeeds`)

	s.Pop()
	require.Equal(t, 1, s.Len(), "Len == 1")

	require.Equal(t, "", s.Lookup("ds"), `Lookup("ds") fails`)

	s.Pop(2)
}
