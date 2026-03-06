package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_omitempty() {
	// The ",omitempty" tag option causes zero-value fields to be omitted
	// from the XML output, matching encoding/xml behavior.
	type Config struct {
		XMLName shim.Name `xml:"config"`
		Host    string    `xml:"host"`
		Port    int       `xml:"port,omitempty"`
		Debug   bool      `xml:"debug,omitempty"`
	}

	// Port and Debug are zero values, so they are omitted.
	c := Config{Host: "localhost"}
	data, err := shim.Marshal(c)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Println(string(data))
	// Output:
	// <config><host>localhost</host></config>
}
