<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: basic test of xs:base64Binary type. -->
  <?spec xpath#id-predefined-types?>
  
  <xsl:param name="q1" select="'01020304a1b2c3d4'"/>
  <xsl:param name="q2" select="'01020304A1B2C3D4'"/>
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="s1" select="xs:base64Binary(xs:hexBinary($q1))"/>
    <xsl:variable name="s2" select="xs:base64Binary(xs:hexBinary($q2))"/>
    <xsl:variable name="s3" select="xs:base64Binary(xs:hexBinary(concat($q1,$q2)))"/>    
    <true><xsl:value-of select="$s1 eq $s2"/></true>
    <false><xsl:value-of select="$s1 eq $s3"/></false>
    <string><xsl:value-of select="string($s3)"/></string>
    <hex><xsl:value-of select="xs:hexBinary(xs:base64Binary(string($s3)))"/></hex>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>