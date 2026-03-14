<xsl:stylesheet version="3.0"
		xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
		xmlns:xs="http://www.w3.org/2001/XMLSchema"
		exclude-result-prefixes="#all">
		
		<!-- Effect of local changes in xsl:version -->

<xsl:param name="s" as="element()*">
  <x>1</x>
  <y>2</y>
  <z>3</z>
</xsl:param>

<xsl:param name="one" select="1" as="xs:integer"/>  

<xsl:template name="main">
    <out>
      <a1 xsl:version="1.0"><xsl:value-of select="($one + $one) instance of xs:double"/></a1>
      <a2 xsl:version="2.0"><xsl:value-of select="($one + $one) instance of xs:double"/></a2>
      <b1 xsl:version="1.0"><xsl:value-of select="$s/j = false()"/></b1>
      <b2 xsl:version="2.0"><xsl:value-of select="$s/j = false()"/></b2>
      <c1 xsl:version="1.0"><xsl:value-of select="$s"/></c1>
      <c2 xsl:version="2.0"><xsl:value-of select="$s"/></c2>
      <d1 xsl:version="1.0"><xsl:sequence select="contains($s, '2')"/></d1>
      <d2 xsl:version="2.0"><xsl:try><xsl:sequence select="contains($s, '2')"/><xsl:catch errors="*">failed</xsl:catch></xsl:try></d2>    
    </out>
</xsl:template>

</xsl:stylesheet>
