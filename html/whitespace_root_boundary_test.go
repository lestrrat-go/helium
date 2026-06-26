package html

import (
	"context"
	"strings"
	"testing"
)

func collectChars(t *testing.T, p Parser, in string) (string, error) {
	t.Helper()
	var sb strings.Builder
	cb := &SAXCallbacks{}
	cb.SetOnCharacters(CharactersFunc(func(ch []byte) error {
		sb.Write(ch)
		return nil
	}))
	err := p.ParseWithSAX(context.Background(), []byte(in), cb)
	return sb.String(), err
}

// TestWhitespaceRootBoundary pins the pre-root / trailing whitespace
// classification across the StripBlanks, SuppressImplied, and content-cap
// configurations that earlier rounds fixed in isolation but regressed against
// one another. All cases must hold simultaneously.
func TestWhitespaceRootBoundary(t *testing.T) {
	// (a) trailing whitespace after explicit empty root must be emitted
	got, err := collectChars(t, NewParser(), "<html></html> \n")
	if err != nil {
		t.Fatalf("(a) err: %v", err)
	}
	if got != " \n" {
		t.Fatalf("(a) trailing ws after </html>: want %q got %q", " \n", got)
	}

	// (b) SuppressImplied + tiny cap: leading space preserved
	got, err = collectChars(t, NewParser().SuppressImplied(true).MaxContentSize(1), "<p> a</p>")
	if err != nil {
		t.Fatalf("(b) err: %v", err)
	}
	if got != " a" {
		t.Fatalf("(b) suppress-implied leading ws: want %q got %q", " a", got)
	}

	// (c) implied body + tiny cap: space+a under implied <body>
	got, err = collectChars(t, NewParser().MaxContentSize(1), "<html> a</html>")
	if err != nil {
		t.Fatalf("(c) err: %v", err)
	}
	if got != " a" {
		t.Fatalf("(c) implied-body leading ws: want %q got %q", " a", got)
	}

	// (d) StripBlanks: leading ws + significant content preserved
	got, err = collectChars(t, NewParser().StripBlanks(true), "<p> &amp;</p>")
	if err != nil {
		t.Fatalf("(d1) err: %v", err)
	}
	if got != " &" {
		t.Fatalf("(d1) stripblanks ws+amp: want %q got %q", " &", got)
	}

	// (e) default-mode over-cap spaces under <p>: soft-cap stream, no hard-fail
	got, err = collectChars(t, NewParser().MaxContentSize(1), "<p>   </p>")
	if err != nil {
		t.Fatalf("(e) over-cap ws under <p> must not hard-fail: %v", err)
	}
	if got != "   " {
		t.Fatalf("(e) soft-cap ws stream: want %q got %q", "   ", got)
	}

	// (f) pre-root whitespace before <html> still dropped
	got, err = collectChars(t, NewParser(), " \n<html><body>x</body></html>")
	if err != nil {
		t.Fatalf("(f) err: %v", err)
	}
	if got != "x" {
		t.Fatalf("(f) pre-root ws must be dropped: want %q got %q", "x", got)
	}
}
