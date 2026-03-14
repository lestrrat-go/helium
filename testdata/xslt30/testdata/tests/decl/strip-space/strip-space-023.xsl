<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <?spec xslt#strip?>

  <xsl:output method="xml" indent="no" />

  <xsl:strip-space elements="*"/>
  
  <xsl:variable name="initial-source" select="."/>

  <xsl:template name="main">
    <xsl:copy-of select="root($initial-source)"/>
  </xsl:template>
</xsl:stylesheet>
