package nsstack_test

import (
	"testing"

	"github.com/lestrrat/helium/internal/stack"
	"github.com/lestrrat/helium/internal/stack/nsstack"
	"github.com/stretchr/testify/assert"
)

func TestNsStack(t *testing.T) {
	s := nsstack.New()
	s.Push("xml", "http://www.w3.org/XML/1998/namespace")
	s.Push("ds", "http://www.w3.org/2000/09/xmldsig#")

	if !assert.Equal(t, 2, s.Len(), "Len == 2") {
		return
	}

	item := s.Lookup("ds")
	if !assert.NotEqual(t, stack.NilItem, item, `Lookup("ds") is not a NilItem`) {
		return
	}

	nsItem := item.(nsstack.Item)
	if !assert.Equal(t, "http://www.w3.org/2000/09/xmldsig#", nsItem.Href, `Lookup("ds") succeeds`) {
		return
	}

	item = s.Lookup("xml")
	if !assert.NotEqual(t, stack.NilItem, item, `Lookup("xm") is not a NilItem`) {
		return
	}

	nsItem = item.(nsstack.Item)
	if !assert.Equal(t, "http://www.w3.org/XML/1998/namespace", nsItem.Href, `Lookup("xml") succeeds`) {
		return
	}

	s.Pop()
	if !assert.Equal(t, 1, s.Len(), "Len == 1") {
		return
	}

	if !assert.Equal(t, stack.NilItem, s.Lookup("ds"), `Lookup("ds") fails`) {
		return
	}

	s.Pop(2)
}