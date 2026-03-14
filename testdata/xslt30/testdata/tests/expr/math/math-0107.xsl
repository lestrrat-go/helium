<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://local.functions/" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all">

  <!-- Simulate Saxon bug 3212 -->

  <xsl:function name="f:parseInt" as="xs:integer?">
    <xsl:param name="in" as="xs:string?"/>
    <xsl:sequence select="xs:integer($in)"/>
  </xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:variable name="range" select="f:parseInt(()) to f:parseInt('123')"/>
      <xsl:if test="42 = $range">
        <matched/>
      </xsl:if>
    </out>
  </xsl:template>
</xsl:stylesheet>
