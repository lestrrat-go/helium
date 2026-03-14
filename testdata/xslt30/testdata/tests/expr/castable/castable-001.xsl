<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
version="2.0">
<!--  XPath section 3.10.3 castable as
-->
<?spec xpath#id-castable?>
<xsl:template match="/">
<out>
<!-- is castable as -->
<a>
<xsl:value-of select="//string castable as xs:string"/>
</a>
<a>
<xsl:value-of select="//boolean castable as xs:boolean"/>
</a>
<a>
<xsl:value-of select="//integer castable as xs:integer"/>
</a>
<a>
<xsl:value-of select="//decimal castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="//float castable as xs:float"/>
</a>
<a>
<xsl:value-of select="//double castable as xs:double"/>
</a>
<a>
<xsl:value-of select="//schemaDuration castable as xs:duration"/>
</a>
<a>
<xsl:value-of select="//dayTimeDuration castable as xs:dayTimeDuration"/>
</a>
<a>
<xsl:value-of select="//yearMonthDuration castable as xs:yearMonthDuration"/>
</a>
<a>
<xsl:value-of select="//dateTime castable as xs:dateTime"/>
</a>
<a>
<xsl:value-of select="//date castable as xs:date"/>
</a>
<a>
<xsl:value-of select="//time castable as xs:time"/>
</a>
<!-- is NOT castable as -->
<b>
<xsl:value-of select="//string castable as xs:boolean"/>
</b>
<b>
<xsl:value-of select="//boolean castable as xs:float"/>
</b>
<b>
<xsl:value-of select="//integer castable as xs:date"/>
</b>
<b>
<xsl:value-of select="//decimal castable as xs:integer"/>
</b>
<b>
<xsl:value-of select="//float castable as xs:date"/>
</b>
<b>
<xsl:value-of select="//double castable as xs:date"/>
</b>
<b>
<xsl:value-of select="//schemaDuration castable as xs:yearMonthDuration"/>
</b>
<b>
<xsl:value-of select="//dayTimeDuration castable as xs:dateTime"/>
</b>
<b>
<xsl:value-of select="//yearMonthDuration castable as xs:dayTimeDuration"/>
</b>
<b>
<xsl:value-of select="//dateTime castable as xs:duration"/>
</b>
<b>
<xsl:value-of select="//date castable as xs:time"/>
</b>
<b>
<xsl:value-of select="//time castable as xs:date"/>
</b>
</out>
</xsl:template>
</xsl:stylesheet>
