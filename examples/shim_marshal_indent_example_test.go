package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_marshal_indent() {
	// shim.MarshalIndent works like encoding/xml.MarshalIndent: it
	// produces indented XML output with the specified prefix and indent strings.
	type Address struct {
		City    string `xml:"city"`
		Country string `xml:"country"`
	}

	addr := Address{City: "Tokyo", Country: "Japan"}
	data, err := shim.MarshalIndent(addr, "", "  ")
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Println(string(data))
	// Output:
	// <Address>
	//   <city>Tokyo</city>
	//   <country>Japan</country>
	// </Address>
}
