<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xsl:output indent="yes"/>

<xsl:param name="s" select="(2, 3 ,4)"/>
<xsl:template match="/">

<!-- cast to a non-atomic type -->
<?spec xpath#id-cast?>
<?error XPST0051?>

  <a><xsl:sequence select="3 cast as xs:anyType"/></a>
</xsl:template>
</xsl:stylesheet>