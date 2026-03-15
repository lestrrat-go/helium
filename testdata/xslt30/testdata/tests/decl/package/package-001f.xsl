<!-- the simplest possible package: visibility has no effect on mode, error XTDE0045 -->
<xsl:package version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  declared-modes="false">

    <xsl:template match="." mode="start" name="somename" visibility="public">
      <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   