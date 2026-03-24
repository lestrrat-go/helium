<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test insert-before() (XPath 2.0) -->
<?spec fo#func-insert-before?>

<xsl:output indent="yes"/>
<xsl:template match="/">
  <xsl:variable name="seq" select="(1,2,3,4,5)"/>
  <out>
    <r1><xsl:value-of select="insert-before($seq, 5, 4.5)" separator=", "/></r1>
    <r2><xsl:value-of select="insert-before($seq, 2, 1.5)" separator=", "/></r2>
    <r3><xsl:value-of select="insert-before($seq, 1, 0.5)" separator=", "/></r3> 
    <r4><xsl:value-of select="insert-before($seq, 10, (9.5, 10.5, 11.5))" separator=", "/></r4>
    <r5><xsl:value-of select="insert-before($seq, -10, (9.5, 10.5, 11.5))" separator=", "/></r5>
    <r6><xsl:value-of select="insert-before($seq, 5, ())" separator=", "/></r6>
    <r7><xsl:value-of select="insert-before($seq, 2, ())" separator=", "/></r7>
    <r8><xsl:value-of select="insert-before($seq, 1, ())" separator=", "/></r8> 
    <r9><xsl:value-of select="insert-before((), 1, $seq)" separator=", "/></r9>  


  </out>
</xsl:template>
 
</xsl:stylesheet>
