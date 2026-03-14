<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#creating-processing-instructions?>

<xsl:template match="/">
<out>
  <xsl:processing-instruction name="p">[[[<xsl:value-of select="'fred'"/>]]]</xsl:processing-instruction>
  <xsl:processing-instruction name="q" select="'[[[', 'fred', ']]]'"/>
</out>
</xsl:template>
        
</xsl:stylesheet>

