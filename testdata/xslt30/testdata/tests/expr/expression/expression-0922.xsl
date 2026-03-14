<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test remove() (XPath 2.0) -->
<?spec fo#func-remove?>

<xsl:output indent="yes"/>
<xsl:template match="/">
  <xsl:variable name="seq" select="(1,2,3,4,5)"/>
  <out>
    <r4><xsl:value-of select="remove($seq, 4)" separator=", "/></r4>
    <r1><xsl:value-of select="remove($seq, 1)" separator=", "/></r1>
    <r0><xsl:value-of select="remove($seq, 0)" separator=", "/></r0> 
    <r9><xsl:value-of select="remove($seq, 9)" separator=", "/></r9>     
  </out>
</xsl:template>
 
</xsl:stylesheet>
