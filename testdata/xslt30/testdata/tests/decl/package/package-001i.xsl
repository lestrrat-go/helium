<!-- the simplest possible implicit package: private declared mode, error XTDE0045 -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">
    
    <xsl:mode name="start" visibility="private"/>

    <xsl:template match="." mode="start" name="somename" visibility="public">
      <not-ok><xsl:value-of select="." /></not-ok>
    </xsl:template>
  
</xsl:stylesheet>