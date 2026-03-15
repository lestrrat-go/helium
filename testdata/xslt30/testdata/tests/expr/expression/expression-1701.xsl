<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test sequence-deep-equal() between nodes -->
<?spec fo#func-deep-equal?>

<xsl:template match="/">
  <out>
    <xsl:variable name="a1" select="(//a)[1]"/>
    <xsl:variable name="a2" select="(//a)[2]"/>
    <xsl:variable name="a3" select="(//a)[3]"/>
    <xsl:variable name="a4" select="(//a)[4]"/>
    <true><xsl:value-of select="deep-equal((), ())"/></true>
    <true><xsl:value-of select="deep-equal(($a1), ($a1))"/></true>
    <true><xsl:value-of select="deep-equal(($a1), ($a2))"/></true>
    <true><xsl:value-of select="deep-equal(($a1, $a2), ($a1, $a2))"/></true>
    <true><xsl:value-of select="deep-equal(($a1, $a2), ($a2, $a1))"/></true>
    <false><xsl:value-of select="deep-equal(($a1,$a2,$a3), ($a2,$a1))"/></false>  
    <false><xsl:value-of select="deep-equal(($a1,$a2), ($a2,$a1,$a3))"/></false>  
    <false><xsl:value-of select="deep-equal(($a1,$a2,$a4), ($a2,$a1,$a3))"/></false> 
  </out>     
</xsl:template>
 
</xsl:stylesheet>
