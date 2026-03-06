package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_marshal() {
	// shim.Marshal works like encoding/xml.Marshal: it serializes a Go
	// struct into XML bytes using struct tags.
	type Person struct {
		XMLName shim.Name `xml:"person"`
		Name    string    `xml:"name"`
		Age     int       `xml:"age"`
	}

	p := Person{Name: "Alice", Age: 30}
	data, err := shim.Marshal(p)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Println(string(data))
	// Output:
	// <person><name>Alice</name><age>30</age></person>
}
