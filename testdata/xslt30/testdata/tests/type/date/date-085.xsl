<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs"
>

<?spec fo#func-subtract-dayTimeDurations?>
<!-- Test subtraction of durations (Priscilla Walmsley bug) -->

   <xsl:template match="/">
    <a><xsl:value-of select='xs:dayTimeDuration("PT51.2S") - xs:dayTimeDuration("PT55.3S")'/></a>

   </xsl:template>

</xsl:stylesheet>
