package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_unmarshal() {
	// shim.Unmarshal works like encoding/xml.Unmarshal: it parses XML
	// and populates a Go struct using struct tags.
	type Item struct {
		XMLName shim.Name `xml:"item"`
		Name    string    `xml:"name"`
		Price   float64   `xml:"price"`
	}

	data := []byte(`<item><name>Widget</name><price>9.99</price></item>`)
	var item Item
	if err := shim.Unmarshal(data, &item); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("Name: %s, Price: %.2f\n", item.Name, item.Price)
	// Output:
	// Name: Widget, Price: 9.99
}
