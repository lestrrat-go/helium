<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:n="http://test.notation.example.com"
exclude-result-prefixes="xs n">

  <!-- PURPOSE: Test casting of string to NOTATION, and from NOTATION to QName.
       Not permitted at run-time in 2.0 (argument must be a string literal). 
       Permitted in 3.0 -->
  <?spec xpath#id-cast?>
  <?error XPTY0004?>
  
  <xsl:import-schema namespace="http://test.notation.example.com" schema-location="simpleNamespaceNotation.xsd"/>
  
  <xsl:param name="q" select="'n:wav'"/>
  
  <xsl:template name="main">
  <out>
    <xsl:variable name="s" select="n:nota($q)" as="xs:NOTATION"/>
    <xsl:value-of select="concat('{', namespace-uri-from-QName(xs:QName($s)), '}', local-name-from-QName(xs:QName($s)))"/>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>