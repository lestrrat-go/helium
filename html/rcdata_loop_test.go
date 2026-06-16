package html_test

import (
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
		t.Run(input, func(t *testing.T) {
			ctx := t.Context()
			done := make(chan struct{})
			go func() {
				defer close(done)
				// We don't care about the result, only that it returns.
				_, _ = html.NewParser().Parse(ctx, []byte(input))
			}()

			// Stoppable timer so the watchdog doesn't linger after the
			// parse returns in the common (fast) case.
			watchdog := time.NewTimer(3 * time.Second)
			defer watchdog.Stop()

			select {
			case <-done:
			case <-watchdog.C:
				t.Fatal("parse hang detected (rcdata invalid end tag infinite loop)")
			}
		})
	}
}
