<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:n="http://test.notation.example.com"
exclude-result-prefixes="xs n">

  <!-- PURPOSE: Test castability of string to NOTATION, and of NOTATION to QName.
       Not permitted at run-time in 2.0 (argument must be a string literal). 
       Permitted in 3.0 -->
  <?spec xpath#id-cast?>
  <?error XPTY0004?>
  
  <xsl:import-schema namespace="http://test.notation.example.com" schema-location="simpleNamespaceNotation.xsd"/>
  
  <xsl:param name="q" select="'n:wav'"/>
  
  <xsl:template name="main">
  <out>
    <a result="{$q castable as n:nota}"/>
    <b result="{$q castable as n:nota-type2}"/>
    <d result="{xs:QName($q) castable as n:nota}"/>
    <e result="{xs:QName($q) castable as n:nota-type2}"/>
    <g result="{n:nota($q) castable as xs:QName}"/>
    <h result="{n:nota($q) castable as n:nota-type2}"/>
    <i result="{n:nota-type2($q) castable as n:nota}"/>
    <j result="{n:nota-type2($q) castable as xs:string}"/>
    <k result="{xs:anyURI($q) castable as xs:QName}"/>
    <l result="{xs:anyURI($q) castable as n:nota}"/>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>