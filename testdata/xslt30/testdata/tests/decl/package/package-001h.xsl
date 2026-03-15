<!-- the simplest possible implicit package: explicit mode in implicit package becomes public -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">
    
    <xsl:mode name="start" />

    <xsl:template match="." mode="start" name="somename" visibility="public">
      <ok><xsl:value-of select="." /></ok>
    </xsl:template>
  
</xsl:stylesheet>