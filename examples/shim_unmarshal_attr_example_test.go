package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_unmarshal_attr() {
	// Struct fields tagged with ",attr" are populated from XML attributes
	// during unmarshal, matching encoding/xml behavior.
	type Image struct {
		XMLName shim.Name `xml:"img"`
		Src     string    `xml:"src,attr"`
		Alt     string    `xml:"alt,attr"`
	}

	data := []byte(`<img src="photo.jpg" alt="A photo"/>`)
	var img Image
	if err := shim.Unmarshal(data, &img); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("Src: %s, Alt: %s\n", img.Src, img.Alt)
	// Output:
	// Src: photo.jpg, Alt: A photo
}
