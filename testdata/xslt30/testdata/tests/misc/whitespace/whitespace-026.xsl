<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#creating-comments?>

<xsl:template match="/">
<out>
  <xsl:comment>[[[<xsl:value-of select="'fred'"/>]]]</xsl:comment>
  <xsl:comment select="'[[[', 'fred', ']]]'"/>
</out>
</xsl:template>
        
</xsl:stylesheet>

