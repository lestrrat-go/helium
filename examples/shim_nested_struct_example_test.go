package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_nested_struct() {
	// Nested structs are marshaled and unmarshaled as nested XML elements,
	// just like encoding/xml.
	type Address struct {
		City    string `xml:"city"`
		Country string `xml:"country"`
	}
	type Person struct {
		XMLName shim.Name `xml:"person"`
		Name    string    `xml:"name"`
		Address Address   `xml:"address"`
	}

	// Marshal
	p := Person{
		Name:    "Bob",
		Address: Address{City: "London", Country: "UK"},
	}
	data, err := shim.Marshal(p)
	if err != nil {
		fmt.Printf("marshal error: %s\n", err)
		return
	}
	fmt.Println(string(data))

	// Unmarshal
	var p2 Person
	if err := shim.Unmarshal(data, &p2); err != nil {
		fmt.Printf("unmarshal error: %s\n", err)
		return
	}
	fmt.Printf("Name: %s, City: %s, Country: %s\n", p2.Name, p2.Address.City, p2.Address.Country)
	// Output:
	// <person><name>Bob</name><address><city>London</city><country>UK</country></address></person>
	// Name: Bob, City: London, Country: UK
}
