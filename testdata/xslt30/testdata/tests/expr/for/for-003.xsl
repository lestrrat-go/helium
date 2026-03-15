<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
<!--  XPath section 3.7 example in note - incorrect version
-->
<?spec xpath#id-for-expressions?>
<xsl:template match="order">
<out>
<xsl:value-of select="sum(for $i in order-item return @price * @qty)"/>
</out>
</xsl:template>
</xsl:stylesheet>
