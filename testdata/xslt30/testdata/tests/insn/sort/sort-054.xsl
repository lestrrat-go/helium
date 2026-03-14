<?spec xslt#sorting?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<xsl:template match="/">

<!-- comparing two xs:anyURI values is now allowed, they are promoted to strings -->

  <xsl:param name="v" as="xs:anyURI">http://www.w3.org/XML</xsl:param>
  <xsl:param name="w" as="xs:anyURI">http://www.w3.org/SVG</xsl:param>

  <a><xsl:sequence select="$v lt $w"/></a>
</xsl:template>
</xsl:stylesheet>