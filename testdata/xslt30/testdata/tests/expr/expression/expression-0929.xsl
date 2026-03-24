<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test reverse() (XPath 2.0) -->
<?spec fo#func-reverse?>

<xsl:template match="/">
  <xsl:variable name="seq" select="(1,2,3,4,5)"/>
  <xsl:variable name="tree"><a><b><c><d/></c></b></a></xsl:variable>
  <out>
     <r1><xsl:value-of select="reverse($seq)" separator=" "/></r1>
     <r2><xsl:value-of select="reverse(6 to 10)" separator=" "/></r2>    
     <r3><xsl:value-of select="reverse(for $i in $tree//d/ancestor::* return name($i))" separator=" "/></r3>          
     <r4><xsl:value-of select="reverse(1)"/></r4>
     <r5><xsl:value-of select="reverse(())"/></r5>     
  </out>
</xsl:template>
 
</xsl:stylesheet>
