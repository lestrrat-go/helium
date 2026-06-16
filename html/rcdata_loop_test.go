package html_test

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
)

// TestRCDATAInvalidEndTagNoHang guards against an infinite loop in
// parseRCDATAContent when a matched-prefix-but-invalid end tag such as
// "</titlex" is encountered. Before the fix these inputs hung forever; a
// watchdog converts a regression into a failure instead of stalling the suite.
func TestRCDATAInvalidEndTagNoHang(t *testing.T) {
	inputs := []string{
		"<title></titlex",
		"<title></titlex>",
		"<textarea></textareax",
	}

	for _, input := range inputs {
		input := input
		t.Run(input, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				// We don't care about the result, only that it returns.
				_, _ = html.NewParser().Parse(context.Background(), []byte(input))
			}()

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("parse hang detected (rcdata invalid end tag infinite loop)")
			}
		})
	}
}
