<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- test a key with a numeric type in a match pattern -->
<!-- for use with key303.xml -->

<xsl:key name="k" match="b" use="string-length(.)"/>



  <xsl:template match="/">
    <out>
     <xsl:apply-templates select="a/b"/> 
    </out>
  </xsl:template>
  
  <xsl:template match="key('k', 3)">
    <three><xsl:value-of select="."/></three>
  </xsl:template>
  
  <xsl:template match="key('k', 4)">
    <four><xsl:value-of select="."/></four>
  </xsl:template>
  
  <xsl:template match="key('k', 5)">
    <five><xsl:value-of select="."/></five>
  </xsl:template>
  
  
 
</xsl:stylesheet>
