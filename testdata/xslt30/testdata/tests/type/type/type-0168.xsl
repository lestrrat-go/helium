<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test casting of string to QName. Not permitted at run-time in 2.0 (argument must be a string literal),
       but permitted in 3.0 (even if the stylesheet says version="2.0"...) -->
  <?spec xpath#id-cast?>
  <?error XPTY0004?>
  
  <xsl:param name="q" select="'pre:thing'"/>
  
  <xsl:template match="/" xmlns:pre="http://magic.namespace.com/uri">
  <out>
    <xsl:variable name="s" select="xs:QName($q)"/>
    <xsl:value-of select="concat('{', namespace-uri-from-QName($s), '}', local-name-from-QName($s))"/>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>