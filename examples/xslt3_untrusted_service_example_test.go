package examples_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

// Example_untrustedService is the pattern to copy into an exposed endpoint that
// transforms an UNTRUSTED stylesheet against an UNTRUSTED source document. It
// layers every bound the caller is responsible for on top of what xslt3 already
// enforces itself. See the "Resource bounds" section of xslt3/README.md for the
// contract split — what xslt3 bounds (per-resource byte cap, recursion depth,
// regex timeout, default-deny I/O, context cancellation) versus what remains the
// caller's job (raw input size, total output size, peak memory, CPU time). A
// context deadline bounds wall-clock time, NOT peak memory, so the raw-input and
// output caps below are not optional.
func Example_untrustedService() {
	// Untrusted request payloads. In a real endpoint these arrive as request
	// bodies / io.Readers of unknown length.
	const untrustedStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">Hello, <xsl:value-of select="name/@who"/>!</xsl:template>
</xsl:stylesheet>`
	const untrustedSource = `<name who="World"/>`

	// 1. Cap raw input size BEFORE parsing. Reject anything larger than a fixed
	//    limit up front, so a huge body never reaches the parser.
	const maxInputBytes = 64 * 1024 // 64 KiB per document
	stylesheetBytes, err := readCapped(bytes.NewReader([]byte(untrustedStylesheet)), maxInputBytes)
	if err != nil {
		fmt.Printf("stylesheet rejected: %s\n", err)
		return
	}
	sourceBytes, err := readCapped(bytes.NewReader([]byte(untrustedSource)), maxInputBytes)
	if err != nil {
		fmt.Printf("source rejected: %s\n", err)
		return
	}

	// 3. Deadline-bearing context. A cancelled/expired ctx aborts compilation and
	//    the transform promptly. Never pass context.Background() for untrusted work.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Hardened parse. helium's default parser is already XXE-blocked,
	//    deny-all-filesystem, and no-network; we do NOT weaken it.
	p := helium.NewParser()

	stylesheetDoc, err := p.Parse(ctx, stylesheetBytes)
	if err != nil {
		fmt.Printf("parse stylesheet error: %s\n", err)
		return
	}
	sourceDoc, err := p.Parse(ctx, sourceBytes)
	if err != nil {
		fmt.Printf("parse source error: %s\n", err)
		return
	}

	// 4. Default-deny external access. We install NO URIResolver / HTTPClient, so
	//    xsl:import, doc(), unparsed-text(), and network reads all stay refused. A
	//    real service grants only a confined resolver rooted at a trusted dir.
	stylesheet, err := xslt3.NewCompiler().Compile(ctx, stylesheetDoc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	// 5 + 6. Cap output through a size-limited writer, and set a small
	//    MaxResourceBytes as defense-in-depth for any resolver-fetched resource
	//    or xsl:analyze-string match enumeration. MaxResourceBytes is PER-RESOURCE,
	//    not a total-output budget — the writer cap is what bounds total output.
	const maxOutputBytes = 1 * 1024 * 1024 // 1 MiB serialized result
	out := &limitedWriter{w: new(bytes.Buffer), remaining: maxOutputBytes}

	err = stylesheet.Transform(sourceDoc).
		MaxResourceBytes(256*1024). // 256 KiB per resolver-fetched resource
		WriteTo(ctx, out)
	if err != nil {
		fmt.Printf("transform error: %s\n", err)
		return
	}

	fmt.Println(out.w.(*bytes.Buffer).String())
	// Output:
	// Hello, World!
}

// readCapped reads at most limit bytes from r and rejects anything larger, so an
// untrusted body of unknown length can never exhaust memory during parsing.
func readCapped(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errInputTooLarge
	}
	return data, nil
}

// limitedWriter caps the total number of bytes written to the wrapped writer,
// returning errOutputTooLarge once the cap would be exceeded. This bounds the
// serialized result size of an output-fanout stylesheet.
type limitedWriter struct {
	w         io.Writer
	remaining int64
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > lw.remaining {
		return 0, errOutputTooLarge
	}
	lw.remaining -= int64(len(p))
	return lw.w.Write(p)
}

var (
	errInputTooLarge  = errors.New("input exceeds maximum allowed size")
	errOutputTooLarge = errors.New("output exceeds maximum allowed size")
)
