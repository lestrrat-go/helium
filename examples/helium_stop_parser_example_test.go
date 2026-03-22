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

	// StopParser lets a SAX callback end parsing early once it has seen enough.
	// This is useful for "scan until found" workloads where building or reading
	// the rest of the document would be wasted work.
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		seen++
		if localname == "stop" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser()
	p.SetSAXHandler(s)

	// Gotcha: stopping the parser this way is treated as an intentional early
	// exit, so Parse returns successfully and you inspect whatever state the
	// handler collected before the stop.
	_, err := p.Parse(context.Background(), []byte(`<root><a/><stop/><b/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(seen)
	// Output:
	// 3
}
