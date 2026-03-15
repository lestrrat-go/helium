<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
<!--  test deep-equal() function -->
<?spec fo#func-deep-equal?>
<xsl:template match="/">
<out>
<true>
<xsl:value-of select="deep-equal(//a[1]/@a, //a[2]/@a)" />
</true>
<false>
<xsl:value-of select="deep-equal(//a[1]/@a, //b[1]/@a)" />
</false>
<false>
<xsl:value-of select="deep-equal(//a[1]/@a, //c[1]/@c)" />
</false>
</out>
</xsl:template>
</xsl:stylesheet>
