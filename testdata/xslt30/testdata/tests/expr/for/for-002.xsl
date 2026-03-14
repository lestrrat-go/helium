<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
<!--  XPath section 3.7 example 2
-->
<?spec xpath#id-for-expressions?>
<xsl:template name="main">
<out>
<xsl:value-of select="for $i in (10, 20),
    $j in (1, 2)
return ($i + $j)" separator=", "/>
</out>
</xsl:template>
</xsl:stylesheet>
