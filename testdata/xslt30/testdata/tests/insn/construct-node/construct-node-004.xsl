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
<xsl:value-of select="xs:duration(//schemaDuration) = //schemaDuration cast as xs:duration"/>
</true>
<true>
<xsl:value-of select="xs:dayTimeDuration(//dayTimeDuration) = //dayTimeDuration cast as xs:dayTimeDuration"/>
</true>
<true>
<xsl:value-of select="xs:yearMonthDuration(//yearMonthDuration) = //yearMonthDuration cast as xs:yearMonthDuration"/>
</true>
</out>
</xsl:template>
</xsl:stylesheet>
