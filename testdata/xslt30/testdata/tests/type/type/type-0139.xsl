<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: basic test of xs:hexBinary type. -->
  <?spec xpath#id-predefined-types?>
  
  <xsl:param name="q1" select="'01020304a1b2c3d4'"/>
  <xsl:param name="q2" select="'01020304A1B2C3D4'"/>
  
  <xsl:template match="/" xmlns:pre="http://magic.namespace.com/uri">
  <out>
    <xsl:variable name="s1" select="xs:hexBinary($q1)"/>
    <xsl:variable name="s2" select="xs:hexBinary($q2)"/>
    <xsl:variable name="s3" select="xs:hexBinary(concat($q1, $q2))"/>    
    <true><xsl:value-of select="$s1 eq $s2"/></true>
    <false><xsl:value-of select="$s1 eq $s3"/></false>
    <string><xsl:value-of select="string($s3)"/></string>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>