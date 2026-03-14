<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test exactly-one(), failure case. -->
  <?spec fo#func-exactly-one?>
  <?error FORG0005?>
<!--Error on line 19
  FORG0005: An empty sequence is not allowed as the first argument of exactly-one()-->
  
  <xsl:param name="p" select="()"/> <!-- force run-time evaluation -->
  <xsl:param name="q" select="()"/>
  <xsl:param name="r" select="(1,2,3)"/>
  
  <xsl:template match="/">
  <out>
    <a><xsl:variable name="a" select="exactly-one($p)" as="xs:integer"/><xsl:value-of select="$a"/></a>
    <b><xsl:variable name="a" select="zero-or-one($p)" as="xs:integer?"/><xsl:value-of select="$a"/></b>  
    <b><xsl:variable name="a" select="zero-or-one($q)" as="xs:integer?"/><xsl:value-of select="$a"/></b>   
    <c><xsl:variable name="a" select="one-or-more($p)" as="xs:integer+"/><xsl:value-of select="$a"/></c>       
    <c><xsl:variable name="a" select="one-or-more($r)" as="xs:integer+"/><xsl:value-of select="$a"/></c>               
  </out>     
  </xsl:template>                             


</xsl:stylesheet>