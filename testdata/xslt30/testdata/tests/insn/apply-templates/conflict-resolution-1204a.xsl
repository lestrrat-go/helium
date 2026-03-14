<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Testing xsl:next-match using import precedence -->
  
<xsl:template match="foo" priority="25">
    <xsl:text>(25)</xsl:text>
    <xsl:next-match/>
</xsl:template>


</xsl:stylesheet>
