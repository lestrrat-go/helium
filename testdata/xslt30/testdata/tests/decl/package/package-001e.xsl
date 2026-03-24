<!-- the simplest possible package: public initial mode -->
<xsl:package version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  declared-modes="false">
    
    <xsl:mode name="start" visibility="public" />

    <xsl:template match="." mode="start">
      <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   