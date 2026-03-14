<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- PURPOSE:   Test subsequence() function with various non-integer parameters -->


<xsl:variable name="s" select="1 to 20"/>
<xsl:variable name="INF" as="xs:double" select="1.0e0 div 0.0e0"/>
<xsl:variable name="NINF" as="xs:double" select="-1.0e0 div 0.0e0"/>
<xsl:variable name="NAN" as="xs:double" select="xs:double('NaN')"/>

<xsl:template match="/">
  <z>
    <a1><xsl:value-of select="subsequence($s, 1.5, 3.2)"/></a1>
    <a2><xsl:value-of select="subsequence($s, -5, 8)"/></a2>  
    <a3><xsl:value-of select="subsequence($s, 5, 30)"/></a3>    
    <a4><xsl:value-of select="subsequence($s, 5, -1)"/></a4>    
    <a5><xsl:value-of select="subsequence($s, $NINF, 5)"/></a5>   
    <a6><xsl:value-of select="subsequence($s, $NINF, $INF)"/></a6>  
    <a7><xsl:value-of select="subsequence($s, 5.2, $INF)"/></a7>  
    <a8><xsl:value-of select="subsequence($s, 5.2, $NAN)"/></a8>   
    <a9><xsl:value-of select="subsequence($s, $NAN, 8)"/></a9>
  </z>                               
</xsl:template>
 
</xsl:stylesheet>
