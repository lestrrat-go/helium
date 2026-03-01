package examples_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

func Example_xinclude_process() {
	// XInclude allows XML documents to include content from other XML files.
	// The main document references an external fragment via <xi:include>.
	const mainSrc = `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="fragment.xml"/>
</doc>`

	// This is the content of the included fragment.
	const fragmentSrc = `<?xml version="1.0"?>
<included>hello from fragment</included>`

	// Create a temporary directory and write both files to it.
	// The parser needs real files on disk because XInclude resolves
	// hrefs relative to the base URI of the including document.
	dir, err := os.MkdirTemp(".", ".tmp-xinclude-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(dir) //nolint:errcheck

	// Convert to absolute path so the XInclude processor can correctly
	// resolve relative hrefs without path-doubling issues.
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

	// parseMain is a helper that parses the main document from disk.
	// SetBaseURI tells the parser the file's location so relative hrefs
	// in xi:include can be resolved.
	parseMain := func() (*helium.Document, error) {
		data, err := os.ReadFile(mainPath)
		if err != nil {
			return nil, err
		}
		p := helium.NewParser()
		p.SetBaseURI(mainPath)
		return p.Parse(data)
	}

	// --- Default behavior: marker nodes ---
	//
	// By default (matching libxml2's behavior), xinclude.Process replaces
	// each xi:include element with a pair of XIncludeStart/XIncludeEnd
	// marker nodes that bracket the included content. These markers
	// serialize as empty <xi:include> elements, allowing applications to
	// track which parts of the tree were included.
	doc, err := parseMain()
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	n, err := xinclude.Process(doc,
		xinclude.WithBaseURI(mainPath),
		xinclude.WithNoBaseFixup(),
	)
	if err != nil {
		fmt.Printf("xinclude error: %s\n", err)
		return
	}
	fmt.Printf("substitutions: %d\n", n)

	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Printf("with markers:\n%s", s)

	// --- WithNoXIncludeMarkers: clean output ---
	//
	// WithNoXIncludeMarkers (equivalent to libxml2's XML_PARSE_NOXINCNODE)
	// removes the xi:include elements entirely after substitution,
	// leaving only the included content in the tree.
	doc, err = parseMain()
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	n, err = xinclude.Process(doc,
		xinclude.WithBaseURI(mainPath),
		xinclude.WithNoBaseFixup(),
		xinclude.WithNoXIncludeMarkers(),
	)
	if err != nil {
		fmt.Printf("xinclude error: %s\n", err)
		return
	}
	fmt.Printf("substitutions: %d\n", n)

	s, err = doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Printf("without markers:\n%s", s)
	// Output:
	// substitutions: 1
	// with markers:
	// <?xml version="1.0"?>
	// <doc xmlns:xi="http://www.w3.org/2001/XInclude">
	//   <xi:include></xi:include><included>hello from fragment</included><xi:include></xi:include>
	// </doc>
	// substitutions: 1
	// without markers:
	// <?xml version="1.0"?>
	// <doc xmlns:xi="http://www.w3.org/2001/XInclude">
	//   <included>hello from fragment</included>
	// </doc>
}
