package xpath3_test

import (
	"testing"
)

func TestFnSortPropagatesAtomizationErrors(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	t.Run("default key", func(t *testing.T) {
		requireXPathErrorCode(t, doc, `sort((1, map{}))`, "FOTY0013")
	})

	t.Run("key function", func(t *testing.T) {
		requireXPathErrorCode(t, doc,
			`sort((1, 2), (), function($x) { if ($x = 1) then $x else map{} })`,
			"FOTY0013")
	})
}
