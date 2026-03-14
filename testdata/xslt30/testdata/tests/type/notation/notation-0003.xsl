<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:n="http://test.notation.example.com"
exclude-result-prefixes="xs n">

  <!-- PURPOSE: Test casting of QName to NOTATION.
       Permitted in 3.0 -->
  <?spec xpath#id-cast?>
  <?error XPTY0004?>
  
  <xsl:import-schema namespace="http://test.notation.example.com" schema-location="simpleNamespaceNotation.xsd"/>
  
  <xsl:param name="q" select="'n:wav'"/>
  
  <xsl:template name="main">
  <out>
    <xsl:variable name="s" select="n:nota(xs:QName($q))" as="xs:NOTATION"/>
    <xsl:variable name="t" select="n:nota-type2(xs:QName($q))" as="xs:NOTATION"/>
    <a><xsl:value-of select="concat('{', namespace-uri-from-QName(xs:QName($s)), '}', local-name-from-QName(xs:QName($s)))"/></a>
    <b><xsl:value-of select="concat('{', namespace-uri-from-QName(xs:QName($t)), '}', local-name-from-QName(xs:QName($s)))"/></b>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>