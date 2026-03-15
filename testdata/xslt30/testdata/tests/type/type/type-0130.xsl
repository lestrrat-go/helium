<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test anyURI datatype. -->
  <?spec fo#casting-to-anyuri?>
  <?spec xpath#id-value-comparisons?>
  
  <xsl:param name="p" select="'example-uri'"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="q" as="xs:anyURI" select="xs:anyURI($p)"/>
    <t><xsl:if test="$q = xs:anyURI('example-uri')">OK</xsl:if></t>;
    <u><xsl:value-of select="$q"/></u>;   
  </out>     
  </xsl:template>                             


</xsl:stylesheet>