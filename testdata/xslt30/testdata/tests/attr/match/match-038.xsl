<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<?spec xslt#patterns?>
    <!-- Purpose: Test the "union" keyword in a pattern (should not be recognized as an operator) -->

<xsl:template match="/ union /*" priority="5">
 <out>WRONG!</out>
</xsl:template>

<xsl:template match="/">
<out>OK!</out>
</xsl:template>

</xsl:stylesheet>
