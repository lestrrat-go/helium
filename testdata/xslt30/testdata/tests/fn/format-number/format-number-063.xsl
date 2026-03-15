<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Test format-number with three arguments -->

<xsl:decimal-format name="a:test" decimal-separator="," grouping-separator="."
    xmlns:a="http://aaa.uri/"/>

<xsl:template match="/">
<o><xsl:value-of select="format-number(12.34, '0.000,00', 'b:test')" xmlns:b="http://aaa.uri/"/></o>
</xsl:template>

</xsl:stylesheet>
