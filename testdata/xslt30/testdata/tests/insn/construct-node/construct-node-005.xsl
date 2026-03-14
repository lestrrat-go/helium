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
<xsl:value-of select="xs:dateTime(//dateTime) = //dateTime cast as xs:dateTime"/>
</true>
<true>
<xsl:value-of select="xs:date(//date) = //date cast as xs:date"/>
</true>
<true>
<xsl:value-of select="xs:time(//time) = //time cast as xs:time"/>
</true>
</out>
</xsl:template>
</xsl:stylesheet>
