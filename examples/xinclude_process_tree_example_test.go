package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

func Example_xinclude_process_tree() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<doc xmlns:xi="http://www.w3.org/2001/XInclude"><wrapper><xi:include href="greeting.xml"/></wrapper></doc>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	wrapper := doc.DocumentElement().FirstChild()
	n, err := xinclude.NewProcessor().
		Resolver(&memoryResolver{
			files: map[string]string{
				"greeting.xml": `<greeting>hello</greeting>`,
			},
		}).
		NoBaseFixup().
		NoXIncludeMarkers().
		ProcessTree(context.Background(), wrapper)
	if err != nil {
		fmt.Printf("xinclude failed: %s\n", err)
		return
	}

	out, err := doc.XMLString()
	if err != nil {
		fmt.Printf("serialize failed: %s\n", err)
		return
	}

	fmt.Println(n)
	fmt.Println(strings.Contains(out, "<greeting>hello</greeting>"))
	// Output:
	// 1
	// true
}
