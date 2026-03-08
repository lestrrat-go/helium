package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_helium_push_parser() {
	// PushParser allows incremental parsing when XML arrives in chunks.
	p := helium.NewParser()
	pp := p.NewPushParser(context.Background())

	if err := pp.Push([]byte(`<root><item>alpha</item>`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}
	if err := pp.Push([]byte(`<item>beta</item></root>`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}

	doc, err := pp.Close()
	if err != nil {
		fmt.Printf("close failed: %s\n", err)
		return
	}

	nodes, err := xpath.Find(context.Background(), doc, `/root/item`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	fmt.Println(len(nodes))
	fmt.Println(string(nodes[0].Content()))
	// Output:
	// 2
	// alpha
}
