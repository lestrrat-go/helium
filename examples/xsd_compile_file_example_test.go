package examples_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_compile_file() {
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="greeting" type="xs:string"/>
</xs:schema>`

	dir, err := os.MkdirTemp(".", ".tmp-xsd-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(dir) //nolint:errcheck

	schemaPath := filepath.Join(dir, "greeting.xsd")
	if err := os.WriteFile(schemaPath, []byte(schemaSrc), 0644); err != nil {
		fmt.Printf("failed to write schema: %s\n", err)
		return
	}

	// CompileFile loads and compiles an XSD schema directly from a file path.
	// This is a convenience over parsing the document yourself and calling Compile.
	schema, err := xsd.CompileFile(schemaPath)
	if err != nil {
		fmt.Printf("failed to compile: %s\n", err)
		return
	}

	doc, err := helium.Parse(context.Background(), []byte(`<greeting>hello</greeting>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	if err := xsd.Validate(context.Background(), doc, schema); err != nil {
		fmt.Println(err)
	}
	// Output:
}
