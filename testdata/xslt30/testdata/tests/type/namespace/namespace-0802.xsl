<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
  xmlns:p="http://dummy/" version="2.0">
  
  <!-- Test use of namespace="" in xsl:element and xsl:attribute -->
  <!-- The spec says the element or attribute should go in the null namespace -->
  <!-- (Bug Saxon 6.2/005) -->

<xsl:template match="/">
<doc>
  <p:part1>
      <xsl:element name="e" namespace="" /> 
      <xsl:element name="p:e" namespace="" /> 
      <e><xsl:attribute name="a" namespace="">1</xsl:attribute></e> 
      <e><xsl:attribute name="p:a" namespace="">2</xsl:attribute></e>
  </p:part1>
  <part2>
      <xsl:element name="e" namespace="" /> 
      <xsl:element name="p:e" namespace="" /> 
      <e><xsl:attribute name="a" namespace="">3</xsl:attribute></e> 
      <e><xsl:attribute name="p:a" namespace="">4</xsl:attribute></e> 
  </part2> 
  <part3 xmlns="http://something/">
      <xsl:element name="e" namespace="" /> 
      <xsl:element name="p:e" namespace="" /> 
      <e><xsl:attribute name="a" namespace="">5</xsl:attribute></e> 
      <e><xsl:attribute name="p:a" namespace="">6</xsl:attribute></e> 
  </part3> 
</doc>
</xsl:template>

</xsl:stylesheet>
