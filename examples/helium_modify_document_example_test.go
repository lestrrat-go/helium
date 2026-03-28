package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_modify_document() {
	// Parse an existing document, then modify it: add a child, remove an
	// attribute, replace a node, and serialize the result.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(
		`<config><db host="old.example.com" port="5432"/><cache/></config>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	root := doc.DocumentElement()

	// Change the host attribute on <db>.
	db := root.FirstChild().(*helium.Element)
	if _, err := db.SetAttribute("host", "new.example.com"); err != nil {
		fmt.Printf("set attribute failed: %s\n", err)
		return
	}

	// Remove the port attribute.
	db.RemoveAttribute("port")

	// Add a new child element to <cache>.
	cache := db.NextSibling().(*helium.Element)
	ttl := doc.CreateElement("ttl")
	if err := ttl.AppendText([]byte("300")); err != nil {
		fmt.Printf("append text failed: %s\n", err)
		return
	}
	if err := cache.AddChild(ttl); err != nil {
		fmt.Printf("add child failed: %s\n", err)
		return
	}

	// Replace <db> with a renamed copy.
	newDB := doc.CreateElement("database")
	if _, err := newDB.SetAttribute("host", "new.example.com"); err != nil {
		fmt.Printf("set attribute failed: %s\n", err)
		return
	}
	if err := db.Replace(newDB); err != nil {
		fmt.Printf("replace failed: %s\n", err)
		return
	}

	out, err := helium.WriteString(root)
	if err != nil {
		fmt.Printf("write failed: %s\n", err)
		return
	}
	fmt.Println(out)
	// Output:
	// <config><database host="new.example.com"/><cache><ttl>300</ttl></cache></config>
}
