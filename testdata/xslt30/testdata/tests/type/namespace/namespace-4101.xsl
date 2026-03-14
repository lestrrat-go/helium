<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:baz1="http://xsl.lotus.com/ns1"
                xmlns:baz2="http://xsl.lotus.com/ns2"
                exclude-result-prefixes="baz1 baz2">

<?spec fo#func-namespace-uri?>
    <!-- Purpose: Test of 'namespace-uri()' function, with hyphenated node name. -->

<xsl:template match="baz2:doc-one">
 <out>
	0 <xsl:value-of select="namespace-uri()"/>:
	1 <xsl:value-of select="namespace-uri(baz1:a-two)" xmlns:baz1="http://xsl.lotus.com/ns1" xmlns:baz2="http://xsl.lotus.com/ns2"/>:
	2 <xsl:value-of select="namespace-uri(baz1:a-two)"/>:
	3 <xsl:value-of select="namespace-uri(baz1:a-two/@attrib1)"/>:
	4 <xsl:value-of select="namespace-uri(baz2:b-three)"/>:
	5 <xsl:value-of select="namespace-uri(baz2:b-three/@baz1:attrib2)"/>:
	6 <xsl:value-of select="namespace-uri(baz2:b-three/c-four)"/>:
	7 <xsl:value-of select="namespace-uri(bogus)"/>:
 </out>
</xsl:template>

</xsl:stylesheet>