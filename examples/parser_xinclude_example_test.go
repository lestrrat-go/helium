package examples_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

// Example_parser_xinclude shows XInclude substitution wired directly into
// parsing via helium.Parser.XInclude. The parser runs the injected processor
// over the document before returning it, so the result already has every
// xi:include expanded — no separate processing call is needed.
func Example_parser_xinclude() {
	const mainSrc = `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="fragment.xml"/>
</doc>`
	const fragmentSrc = `<?xml version="1.0"?>
<included>hello from fragment</included>`

	// XInclude resolves hrefs relative to the including document's location, so
	// write both files to a temp dir and parse the main file from disk.
	dir, err := os.MkdirTemp(".", ".tmp-xinclude-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(dir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("failed to get abs path: %s\n", err)
		return
	}

	mainPath := filepath.Join(absDir, "main.xml")
	fragPath := filepath.Join(absDir, "fragment.xml")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		fmt.Printf("failed to write: %s\n", err)
		return
	}
	if err := os.WriteFile(fragPath, []byte(fragmentSrc), 0644); err != nil {
		fmt.Printf("failed to write: %s\n", err)
		return
	}

	// Configure an XInclude processor. It denies all filesystem access unless a
	// resolver is supplied; helium.PermissiveFS() opens any OS path (these files
	// live in a temp dir). NoXIncludeMarkers drops the bracketing xi:include
	// marker nodes so only the included content remains.
	proc := xinclude.NewProcessor().
		Resolver(xinclude.NewFSResolver(helium.PermissiveFS())).
		BaseURI(mainPath).
		NoBaseFixup().
		NoXIncludeMarkers()

	// Inject the processor: ParseFile returns a document with includes expanded.
	doc, err := helium.NewParser().XInclude(proc).ParseFile(context.Background(), mainPath)
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	s, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)
	// Output:
	// <?xml version="1.0"?>
	// <doc xmlns:xi="http://www.w3.org/2001/XInclude">
	//   <included>hello from fragment</included>
	// </doc>
}
