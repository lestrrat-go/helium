<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: test format-number() with grouping separator -->

<xsl:template match="/">
<out>
<xsl:value-of select="format-number(123456789.34, '#,###.##')"/>
</out>
</xsl:template>

</xsl:stylesheet>
