package examples_test

import (
	"fmt"
	"os"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
)

func Example_schematron_compile_file() {
	f, err := os.CreateTemp("", "helium-schematron-*.sch")
	if err != nil {
		fmt.Printf("create temp file failed: %s\n", err)
		return
	}
	defer os.Remove(f.Name()) //nolint:errcheck

	if _, err := f.WriteString(`<schema xmlns="http://www.ascc.net/xml/schematron"><pattern name="book-check"><rule context="book"><assert test="title">title is required</assert></rule></pattern></schema>`); err != nil {
		fmt.Printf("write temp file failed: %s\n", err)
		return
	}
	if err := f.Close(); err != nil {
		fmt.Printf("close temp file failed: %s\n", err)
		return
	}

	schema, err := schematron.CompileFile(f.Name())
	if err != nil {
		fmt.Printf("compile failed: %s\n", err)
		return
	}

	doc, err := helium.Parse([]byte(`<book><title>Helium</title></book>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	if err := schematron.Validate(doc, schema, schematron.WithFilename("doc.xml")); err != nil {
		fmt.Println(err)
	}
	// Output:
}
