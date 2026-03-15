<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <out>   
      <xsl:value-of select="banana"/>
      <xsl:on-empty select="'apple'"/> 
  </out>
</xsl:template>

</xsl:stylesheet>
