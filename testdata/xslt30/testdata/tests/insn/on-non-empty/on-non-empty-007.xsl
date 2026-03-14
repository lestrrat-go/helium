<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:param name="z" select="0"/>

<xsl:template match="/">
  <out>
    <xsl:variable name="x" select="21 + $z"/>
    <xsl:on-non-empty select="$x"/>
    <xsl:copy-of select="/comment()"/>
    <xsl:variable name="x" select="23 + $z"/>    
    <xsl:on-non-empty select="$x"/>
    <xsl:copy-of select="/processing-instruction()"/>
    <in/> 
  </out>
</xsl:template>

</xsl:stylesheet>
