<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
<!-- Test use of a range variable in a pattern (XSLT 2.0) -->

<xsl:template match="*">
<xsl:copy><xsl:apply-templates/></xsl:copy>
</xsl:template>
  
<xsl:template match="*[some $x in child::* satisfies name($x) = name(.)]">
<xsl:copy><xsl:attribute name="recursive">yes</xsl:attribute>
<xsl:apply-templates/></xsl:copy>
</xsl:template>

</xsl:stylesheet>
