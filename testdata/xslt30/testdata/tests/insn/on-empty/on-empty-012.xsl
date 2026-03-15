<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

      <xsl:param name="banana">
        <xsl:copy-of select="/banana"/>
      </xsl:param>

<xsl:template match="/">
  <out>   
      <xsl:sequence select="$banana"/>
      <xsl:on-empty select="'apple'"/> 
  </out>
</xsl:template>

</xsl:stylesheet>
