package examples_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

type exampleXPath3MemoryResolver struct {
	files map[string]string
}

func (r exampleXPath3MemoryResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	content, ok := r.files[uri]
	if !ok {
		return nil, fmt.Errorf("not found: %s", uri)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func Example_xpath3_resolver_options() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	compiled, err := xpath3.NewCompiler().Compile(`upper-case(unparsed-text($file))`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("mem://docs/").
		URIResolver(exampleXPath3MemoryResolver{
			files: map[string]string{
				"mem://docs/greeting.txt": "hello from resolver",
			},
		}).
		Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
			"file": xpath3.SingleString("greeting.txt"),
		})).
		Evaluate(context.Background(), compiled, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}

	s, ok := r.IsString()
	if !ok {
		fmt.Println("unexpected non-string result")
		return
	}
	fmt.Println(s)
	// Output:
	// HELLO FROM RESOLVER
}
