package examples_test

import (
	"context"
	"fmt"
	"math/big"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xslt3_call_function() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://example.com/fn">
  <xsl:function name="f:double" as="xs:integer"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    visibility="public">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="$n * 2"/>
  </xsl:function>
</xsl:stylesheet>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	// CallFunction invokes a named public function directly.
	// The function name uses Clark notation: {namespace}local-name.
	// Arguments are passed as positional xpath3.Sequence values.

	// Build the argument: an xs:integer with value 21.
	arg := xpath3.SingleAtomic(xpath3.AtomicValue{
		TypeName: xpath3.TypeInteger,
		Value:    big.NewInt(21),
	})

	resultDoc, err := stylesheet.
		CallFunction(helium.ClarkName("http://example.com/fn", "double"), arg).
		Do(ctx)
	if err != nil {
		fmt.Printf("call-function error: %s\n", err)
		return
	}

	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("serialize error: %s\n", err)
		return
	}

	fmt.Println(out)
	// Output:
	// 42
}
