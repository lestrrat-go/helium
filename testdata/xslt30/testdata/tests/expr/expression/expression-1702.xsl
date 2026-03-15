<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test deep-equal() between simple values -->
<?spec fo#func-deep-equal?>

<xsl:template match="/">
  <out>
    <xsl:variable name="a1" select="3"/>
    <xsl:variable name="a2" select="3.0"/>
    <xsl:variable name="a3" select="'3'"/>
    <xsl:variable name="a4" select="current-dateTime()"/>
    <true><xsl:value-of select="deep-equal((), ())"/></true>
    <true><xsl:value-of select="deep-equal(($a1), ($a1))"/></true>
    <true><xsl:value-of select="deep-equal(($a1), ($a2))"/></true>
    <true><xsl:value-of select="deep-equal(($a1, $a2), ($a1, $a2))"/></true>
    <true note="false in 7.4, compares number and string"><xsl:value-of select="deep-equal(($a1, $a2, $a3, $a4), ($a2, $a3, $a1, $a4))"/></true>
    <false><xsl:value-of select="deep-equal(($a1,$a4), ($a2, $a1))"/></false>  
    <false><xsl:value-of select="deep-equal(($a1,$a2), ($a2,$a1,$a3))"/></false>  
    <false><xsl:value-of select="deep-equal(($a1,$a2,$a4), ($a2,$a1,$a3))"/></false> 
  </out>     
</xsl:template>
 
</xsl:stylesheet>
