<xsl:stylesheet
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:strip-space elements="document"/>
  <?spec xslt#strip?>

  <xsl:template match="document">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="section">
    <xsl:text>Position:</xsl:text><xsl:value-of select="position()"/>
    <xsl:text>,Last:</xsl:text><xsl:value-of select="last()"/><xsl:text>
</xsl:text>
</xsl:template>

</xsl:stylesheet>

