<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-numeric-divide?>
<!-- PURPOSE:  Test of division between decimal values. -->
<!-- Because numeric precision is implementation-defined, we round the result to 6 places (which most
     implementations are likely to support, hopefully -->
<xsl:template match="doc">
  <out>
    <xsl:value-of select="10.0 div 5"/>;
    <xsl:value-of select="10.0 div 5.0"/>;   
    <xsl:value-of select="10.00 div 5.0"/>;    
    <xsl:value-of select="round-half-to-even(10.0 div 3.0, 6)"/>;     
    <xsl:value-of select="round-half-to-even(10.0 div 3, 6)"/>;         
    <xsl:value-of select="3.0 div 10.0"/>;  
    <xsl:value-of select="0.0001 div 0.0001"/>;  
    <xsl:value-of select="round-half-to-even(0.0001 div 0.3, 6)"/>;                       
  </out>
  </xsl:template>
 
</xsl:stylesheet>
