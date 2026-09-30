package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_character_data() {
	const src = `<!DOCTYPE price [<!ENTITY cents "99">]>
<price>12.<!-- cents follow -->&cents;<note>USD</note></price>`

	// The default parser keeps &cents; as an entity reference node.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}
	root := doc.DocumentElement()

	// CharacterData returns only the text the element holds directly, with the
	// entity reference expanded. The comment and the <note> child add nothing.
	fmt.Println(helium.CharacterData(root))

	// Content, by contrast, also includes the text of descendant elements.
	fmt.Println(string(root.Content()))
	// Output:
	// 12.99
	// 12.99USD
}
