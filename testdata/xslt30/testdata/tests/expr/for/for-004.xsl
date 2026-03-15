<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
<!--  XPath section 3.7 example in note - correct version
-->
<?spec xpath#id-for-expressions?>
<xsl:template match="order">
<out>
<xsl:value-of select="format-number(
sum(for $i in order-item return $i/@price * $i/@qty),
'0.00')"/>
</out>
</xsl:template>
</xsl:stylesheet>
