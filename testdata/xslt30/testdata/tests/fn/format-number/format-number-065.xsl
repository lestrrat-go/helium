<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: test duplicate unnamed xsl:decimal-format elements: OK because compatible -->
<xsl:decimal-format decimal-separator="," grouping-separator="."/>
<xsl:decimal-format decimal-separator="," grouping-separator="."/>

<xsl:template match="/">
<out>
<xsl:value-of select="format-number(12.34, '000,000')"/>
</out>
</xsl:template>

</xsl:stylesheet>
