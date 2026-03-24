<!-- the simplest possible implicit package: though the visibility attribute here has no effect, the implicit mode is public, bug #30399 -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">

    <xsl:template match="." mode="start" name="somename" visibility="public">
      <ok><xsl:value-of select="." /></ok>
    </xsl:template>
  
</xsl:stylesheet>