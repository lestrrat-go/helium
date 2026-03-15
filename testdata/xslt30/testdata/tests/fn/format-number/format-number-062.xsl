<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: test format-number() with two arguments -->

<xsl:template match="/">
<o><xsl:value-of select="format-number(12.34, '##.##')"/></o>
</xsl:template>

</xsl:stylesheet>
