<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xsl:output indent="yes"/>

<xsl:template match="/">

<!-- compare untyped atomic to integer -->
<?spec xpath#id-value-comparisons?> 
<?error XPTY0004?>

  <xsl:param name="v">72</xsl:param>

  <a><xsl:sequence select="$v gt 70"/></a>
</xsl:template>
</xsl:stylesheet>