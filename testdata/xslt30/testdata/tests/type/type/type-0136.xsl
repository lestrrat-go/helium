<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test exactly-one(), zero-or-one(), one-or-more(), success case. -->
  <?spec fo#func-exactly-one?>
  <?spec fo#func-zero-or-one?>
  <?spec fo#func-one-or-more?>    
  
  <xsl:param name="p" select="1"/> <!-- force run-time evaluation -->
  <xsl:param name="q" select="()"/>
  <xsl:param name="r" select="(1,2,3)"/>
  
  <xsl:template match="/">
  <out>
    <a><xsl:variable name="a" select="exactly-one($p)" as="xs:integer"/><xsl:sequence select="$a"/></a>
    <b><xsl:variable name="a"  select="zero-or-one($p)" as="xs:integer?"/><xsl:sequence select="$a"/></b>  
    <b><xsl:variable name="a"  select="zero-or-one($q)" as="xs:integer?"/><xsl:sequence select="$a"/></b>  
    <c><xsl:variable name="a"  select="one-or-more($p)" as="xs:integer+"/><xsl:sequence select="$a"/></c>       
    <c><xsl:variable name="a"  select="one-or-more($r)" as="xs:integer+"/><xsl:sequence select="$a"/></c>               
  </out>     
  </xsl:template>                             


</xsl:stylesheet>