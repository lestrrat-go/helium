<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xsl:output indent="yes"/>

<xsl:template match="/">

<!-- error: compare an unordered type -->
<?spec xpath#id-value-comparisons?> 
<?error XPTY0004?>

  <xsl:param name="v" as="xs:gYear">2004</xsl:param>
  <xsl:param name="w" as="xs:gYear">2005</xsl:param>

  <a><xsl:sequence select="$v lt $w"/></a>
</xsl:template>
</xsl:stylesheet>