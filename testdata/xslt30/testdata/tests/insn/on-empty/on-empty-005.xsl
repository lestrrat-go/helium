<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:strip-space elements="*"/>

<xsl:template match="/">
  <out>
    <xsl:copy-of select="/*"/>   
    <xsl:on-empty select="23"/>
  </out>
</xsl:template>

</xsl:stylesheet>
