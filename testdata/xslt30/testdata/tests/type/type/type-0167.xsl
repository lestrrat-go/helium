<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xsl:output indent="yes"/>

<xsl:template match="/">

<!-- check that decimal does not allow an exponent -->
<?spec fo#casting-to-numerics?>
<?error FORG0001?>

  <xsl:variable name="v" as="xs:decimal" select="xs:decimal('93.7e5')"/>

  <a><xsl:sequence select="$v"/></a>
</xsl:template>
</xsl:stylesheet>