package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestTransformSerializationParamsQName drives fn:transform with a serialized
// delivery format and a serialization-params map carrying QName-valued
// (cdata-section-elements, suppress-indentation) and map-valued
// (use-character-maps) parameters, mirroring QT3 cases fn-transform-65,
// fn-transform-66, and fn-transform-67. Each parameter must be applied to the
// serialized principal result, unioned (cdata-section-elements /
// suppress-indentation) or merged with override precedence (use-character-maps)
// over the stylesheet's own xsl:output values.
func TestTransformSerializationParamsQName(t *testing.T) {
	sourceDoc := helium.NewDefaultDocument()

	// fn-transform-65: serialization-params cdata-section-elements. The stylesheet
	// declares cdata-section-elements='b my:b'; the param adds (my:c, c). The
	// serialized output wraps the content of b, my:b, my:c, and c in CDATA, but
	// leaves d as ordinary text.
	t.Run("CDATASectionElements", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='2.0'>
            <xsl:output cdata-section-elements='b my:b'/>
            <xsl:template name='main'>
              <my:a>
                <my:b>green</my:b>
                <my:c>blue</my:c>
                <b>red</b>
                <c>pink</c>
                <d>black</d>
              </my:a>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'cdata-section-elements': (QName('http://www.w3.org/fots/fn/transform/myfunctions','c'), QName('', 'c'))}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), "[CDATA[green]]")
             and contains($result("output"), "[CDATA[blue]]")
             and contains($result("output"), "[CDATA[red]]")
             and contains($result("output"), "[CDATA[pink]]")
             and contains($result("output"), "<d>black</d>"))`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})

	// fn-transform-66: serialization-params suppress-indentation. The stylesheet
	// declares indent='yes' suppress-indentation='b my:b'; the param adds
	// (my:c, c). The serialized output keeps b, my:b, my:c, and c un-indented but
	// indents d.
	t.Run("SuppressIndentation", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='3.0'>
            <xsl:output indent='yes' suppress-indentation='b my:b'/>
            <xsl:template name='main'>
              <my:a>
                <my:b><t>green</t></my:b>
                <my:c><t>blue</t></my:c>
                <b><t>red</t></b>
                <c><t>pink</t></c>
                <d><t>black</t></d>
              </my:a>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'suppress-indentation': (QName('http://www.w3.org/fots/fn/transform/myfunctions','c'), QName('', 'c'))}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), "><t>green</t><")
             and contains($result("output"), "><t>blue</t><")
             and contains($result("output"), "><t>red</t><")
             and contains($result("output"), "><t>pink</t><")
             and matches($result("output"), ">\s+<t>black</t>\s+<"))`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})

	// fn-transform-67: serialization-params use-character-maps. The stylesheet's
	// map-one maps '-'→'(hyphen)' and '*'→'(asterisk)'; the param map remaps
	// '*'→'(star)'. The serialized output merges the two, with the param winning
	// for '*'.
	t.Run("UseCharacterMaps", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='2.0'>
            <xsl:output use-character-maps='map-one'/>
            <xsl:character-map name='map-one'>
              <xsl:output-character character='-' string='(hyphen)'/>
              <xsl:output-character character='*' string='(asterisk)'/>
            </xsl:character-map>
            <xsl:template name='main'>
              <out>a-b*c</out>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'use-character-maps': map{'*':'(star)'}}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), ">a(hyphen)b(star)c</out>"))`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})
}
