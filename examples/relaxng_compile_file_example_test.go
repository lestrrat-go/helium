package examples_test

import (
	"context"
	"fmt"
	"os"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_compile_file() {
	// CompileFile is the convenient RELAX NG entry point when the grammar
	// already lives on disk, for example as part of an application's fixtures
	// or configuration bundle.
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

	grammar, err := relaxng.NewCompiler().CompileFile(context.Background(), f.Name())
	if err != nil {
		fmt.Printf("compile failed: %s\n", err)
		return
	}

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	// Create a validator from the compiled grammar. Label sets the
	// document name used in error messages (it does not read from disk).
	// Successful validation is intentionally quiet: nil means the document
	// matched the grammar.
	v := relaxng.NewValidator(grammar).
		Label("doc.xml")

	if err := v.Validate(context.Background(), doc); err != nil {
		fmt.Println(err)
	}
	// Output:
}
