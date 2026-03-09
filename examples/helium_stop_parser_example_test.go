package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
)

func Example_helium_stop_parser() {
	var seen int
	s := sax.New()
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		seen++
		if localname == "stop" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser()
	p.SetSAXHandler(s)
	_, err := p.Parse(context.Background(), []byte(`<root><a/><stop/><b/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(seen)
	// Output:
	// 3
}
