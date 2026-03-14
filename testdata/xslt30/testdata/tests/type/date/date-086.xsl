<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs"
>

<?spec fo#func-subtract-dateTimes-yielding-dayTimeDuration?>
<!-- Test subtraction of times with/without timezone -->

   <xsl:template match="/">
    <a><xsl:value-of select="xs:dateTime('2005-01-01T15:00:00') - current-dateTime()
                                 gt xs:dayTimeDuration('PT0S')"/></a>

   </xsl:template>

</xsl:stylesheet>
