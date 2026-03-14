<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
   
   <?spec fo#func-numeric-divide?>
<!-- PURPOSE:  Determine whether mod returns +0 or -0. -->
<xsl:template match="doc">
  <out>
    <xsl:choose>
    <xsl:when test="number(system-property('xsl:version')) = 1.0">
        <xsl:value-of select="1 div (2 mod 2)"/>;
        <xsl:value-of select="1 div (2 mod -2)"/>;   
        <xsl:value-of select="1 div (-2 mod 2)"/>;
        <xsl:value-of select="1 div (-2 mod -2)"/>;   
    </xsl:when>
    <xsl:otherwise>
        <xsl:value-of select="1 div (2e0 mod 2)"/>;
        <xsl:value-of select="1 div (2e0 mod -2)"/>;   
        <xsl:value-of select="1 div (-2e0 mod 2)"/>;
        <xsl:value-of select="1 div (-2e0 mod -2)"/>;   
    </xsl:otherwise>
    </xsl:choose>                  
  </out>
  </xsl:template>
 
</xsl:stylesheet>
