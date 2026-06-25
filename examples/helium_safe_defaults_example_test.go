package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_safe_defaults() {
	// NewParser() is secure by default and needs no extra configuration to
	// handle input from untrusted sources: external entity and DTD loading is
	// blocked (XXE), no filesystem is exposed, network access is forbidden, and
	// element nesting depth is capped at 256.
	//
	// To deliberately load external resources from a trusted source, opt back in
	// explicitly, e.g.:
	//
	//	helium.NewParser().
	//		BlockXXE(false).
	//		LoadExternalDTD(true).
	//		FS(helium.PermissiveFS()). // or a confined fs.FS rooted at a trusted dir
	//		Parse(ctx, data)
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<order id="42"><item>book</item></order>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	s, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <order id="42"><item>book</item></order>
}
