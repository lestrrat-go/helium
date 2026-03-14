<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://f.com/" exclude-result-prefixes="f xs" xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- test arithmetic on big integers -->

  <xsl:function name="f:mangle" as="xs:integer">
    <xsl:param name="in" as="xs:integer"/>
    <xsl:variable name="a" select="$in * 200001"/>
    <xsl:variable name="b" select="$a + 862"/>
    <xsl:sequence
      select="if (($b div 2) gt 99999999999999999999999999999999999999999) then $b else f:mangle($b)"
    />
  </xsl:function>
  <xsl:template match="/">
    <a>
      <xsl:copy-of select="f:mangle(199)"/>
    </a>
  </xsl:template>
</xsl:stylesheet>
