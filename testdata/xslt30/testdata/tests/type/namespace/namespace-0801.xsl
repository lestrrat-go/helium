<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
  xmlns="http://dummy/" version="2.0">

<xsl:template match="/">
  <xsl:variable name="test">
    <xsl:copy-of select="//XXX"/>
    <xsl:copy-of select="//*[local-name()='YYY']"/>
  </xsl:variable>
  <out>
  <Variable>
    <xsl:copy-of select="$test"/>
  </Variable>
  <Direct>
    <xsl:copy-of select="//XXX"/>
    <xsl:copy-of select="//*[local-name()='YYY']"/>
  </Direct>
  </out>
</xsl:template>

</xsl:stylesheet>
