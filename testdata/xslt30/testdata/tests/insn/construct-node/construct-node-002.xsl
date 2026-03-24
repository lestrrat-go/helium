<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
version="2.0">
<!--  XPath section 3.10.4 constructor functions
-->
<?spec xpath#id-castable?>
<xsl:template match="/">
<out>
<true>
<xsl:value-of select="xs:integer(//integer) = //integer cast as xs:integer"/>
</true>
<true>
<xsl:value-of select="xs:decimal(//decimal) = //decimal cast as xs:decimal"/>
</true>
<true>
<xsl:value-of select="xs:float(//float) = //float cast as xs:float"/>
</true>
<true>
<xsl:value-of select="xs:double(//double) = //double cast as xs:double"/>
</true>
</out>
</xsl:template>
</xsl:stylesheet>
