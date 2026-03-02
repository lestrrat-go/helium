package examples_test

import (
	"fmt"
	"os"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_compile_file() {
	f, err := os.CreateTemp("", "helium-relaxng-*.rng")
	if err != nil {
		fmt.Printf("create temp file failed: %s\n", err)
		return
	}
	defer os.Remove(f.Name()) //nolint:errcheck

	if _, err := f.WriteString(`<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="book"><element name="title"><text/></element></element></start></grammar>`); err != nil {
		fmt.Printf("write temp file failed: %s\n", err)
		return
	}
	if err := f.Close(); err != nil {
		fmt.Printf("close temp file failed: %s\n", err)
		return
	}

	grammar, err := relaxng.CompileFile(f.Name())
	if err != nil {
		fmt.Printf("compile failed: %s\n", err)
		return
	}

	doc, err := helium.Parse([]byte(`<book><title>Helium</title></book>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	if err := relaxng.Validate(doc, grammar, relaxng.WithFilename("doc.xml")); err != nil {
		fmt.Println(err)
	}
	// Output:
}
