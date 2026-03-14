<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:baz1="http://xsl.lotus.com/ns1"
                xmlns:baz2="http://xsl.lotus.com/ns2">

<?spec fo#func-namespace-uri?>
    <!-- Purpose: Test of 'namespace-uri' function. -->

<xsl:template match="baz2:doc">
	<out>
		<xsl:value-of select="namespace-uri(x)" xmlns:baz1="http://xsl.lotus.com/ns1" xmlns:baz2="http://xsl.lotus.com/ns2"/>
	</out>
</xsl:template>

</xsl:stylesheet>