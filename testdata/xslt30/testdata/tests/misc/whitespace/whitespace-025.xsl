<xsl:stylesheet version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:foo="http://whatever">
  
  <?spec xslt#creating-text-nodes?>

  <xsl:function name="foo:write1" as="xs:string*">
    <xsl:param name="x" as="xs:string"/>
    <xsl:value-of select="$x"/>
  </xsl:function>

  <xsl:function name="foo:write2" as="text()*">
    <xsl:param name="x" as="xs:string"/>
    <xsl:text>'</xsl:text><xsl:value-of select="$x"/><xsl:text>'</xsl:text>
  </xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:text>'</xsl:text>
      <xsl:value-of select="foo:write1('test')"/>
      <xsl:text>'</xsl:text>
      <xsl:value-of select="foo:write2('test')"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
