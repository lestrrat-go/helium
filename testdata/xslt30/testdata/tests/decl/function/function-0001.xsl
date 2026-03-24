<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <xsl:template match="/" name="main">
    <out>
      <a>
        <xsl:value-of select="format-integer(1 ,'0')"/>
      </a>
      <b>
        <xsl:value-of select="format-integer(11 ,'#0')"/>
      </b>
      <c>
        <xsl:value-of select="format-integer(1230 ,'#,000')"/>
      </c>
    </out>
  </xsl:template>
</xsl:stylesheet>
