package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestXPathErrorErrorNilReceiver(t *testing.T) {
	var err *xpath3.XPathError
	require.Equal(t, "<nil XPathError>", err.Error())
}
