<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test resolve-QName, success case. -->
  <?spec fo#func-resolve-QName?>
  <?spec fo#func-namespace-uri-from-QName?>
  <?spec fo#func-local-name-from-QName?>
  
  <xsl:param name="p" select="'thing'"/> <!-- force run-time evaluation -->
  <xsl:param name="q" select="'pre:thing'"/>

  <xsl:variable name="doc">
    <e xmlns:pre="http://magic.namespace.com/uri"/>
  </xsl:variable>
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="p" select="resolve-QName($p, $doc/e)" as="xs:QName"/>
    <xsl:variable name="q" select="resolve-QName($q, $doc/e)" as="xs:QName"/>    
    <a><xsl:value-of select="(namespace-uri-from-QName($p), local-name-from-QName($p))" separator=","/></a>
    <b><xsl:value-of select="(namespace-uri-from-QName($q), local-name-from-QName($q))" separator=","/></b>  
  </out>     
  </xsl:template>                             


</xsl:stylesheet>