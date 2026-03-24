<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
version="2.0">
<!--  XPath section 3.10.3 castable as
-->
<?spec xpath#id-castable?>
<xsl:template match="/">
<xsl:variable name="string" select="//string cast as xs:string"/>
<xsl:variable name="boolean" select="//boolean cast as xs:boolean"/>
<xsl:variable name="integer" select="//integer cast as xs:integer"/>
<xsl:variable name="decimal" select="//decimal cast as xs:decimal"/>
<xsl:variable name="float" select="//float cast as xs:float"/>
<xsl:variable name="double" select="//double cast as xs:double"/>
<xsl:variable name="schemaDuration" select="//schemaDuration cast as xs:duration"/>
<xsl:variable name="dayTimeDuration" select="//dayTimeDuration cast as xs:dayTimeDuration"/>
<xsl:variable name="yearMonthDuration" select="//yearMonthDuration cast as xs:yearMonthDuration"/>
<xsl:variable name="dateTime" select="//dateTime cast as xs:dateTime"/>
<xsl:variable name="date" select="//date cast as xs:date"/>
<xsl:variable name="time" select="//time cast as xs:time"/>
<out>
<!-- is castable as -->
<a>
<xsl:value-of select="$float castable as xs:float"/>
</a>
<a>
<xsl:value-of select="$double castable as xs:float"/>
</a>
<a>
<xsl:value-of select="$decimal castable as xs:float"/>
</a>
<a>
<xsl:value-of select="$integer castable as xs:float"/>
</a>
<a>
<xsl:value-of select="$boolean castable as xs:float"/>
</a>
<a>
<xsl:value-of select="$double castable as xs:double"/>
</a>
<a>
<xsl:value-of select="$float castable as xs:double"/>
</a>
<a>
<xsl:value-of select="$decimal castable as xs:double"/>
</a>
<a>
<xsl:value-of select="$integer castable as xs:double"/>
</a>
<a>
<xsl:value-of select="$boolean castable as xs:double"/>
</a>
<a>
<xsl:value-of select="$decimal castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="$float castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="$double castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="$integer castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="$boolean castable as xs:decimal"/>
</a>
<a>
<xsl:value-of select="$decimal castable as xs:integer"/>
</a>
<a>
<xsl:value-of select="$float castable as xs:integer"/>
</a>
<a>
<xsl:value-of select="$double castable as xs:integer"/>
</a>
<a>
<xsl:value-of select="$integer castable as xs:integer"/>
</a>
<a>
<xsl:value-of select="$boolean castable as xs:integer"/>
</a>
<!-- is NOT castable as -->
<b>
<xsl:value-of select="$string castable as xs:float"/>
</b>
<b>
<xsl:value-of select="$schemaDuration castable as xs:float"/>
</b>
<b>
<xsl:value-of select="$dateTime castable as xs:float"/>
</b>
</out>
</xsl:template>
</xsl:stylesheet>
