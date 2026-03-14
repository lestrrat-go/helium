<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">
                
<?spec fo#func-seconds-from-duration?> 
                      
  <xsl:template match="/" >
    <xsl:variable name="secs" select="xs:duration('PT15S')"/>
    <xsl:variable name="t" select="seconds-from-duration($secs)" as="xs:decimal"/>
    <d><xsl:value-of select="$t"/></d>
  </xsl:template>

</xsl:stylesheet>
