<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
<!--  XPath section 3.7 example 1
-->
<?spec xpath#id-for-expressions?>
<xsl:template match="/">
<out>
<xsl:sequence select="for $a in distinct-values(/bib/book/author)
return ((/bib/book/author[. = $a])[1], /bib/book[author = $a]/title)"/>
</out>
</xsl:template>
</xsl:stylesheet>
