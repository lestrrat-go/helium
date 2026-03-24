<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test QName comparison. -->
  <?spec fo#func-QName-equal?>
  
  <xsl:param name="p" select="'example-uri'"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="q" as="xs:QName" select="QName($p, 'example-lname')"/>
    <t><xsl:if test="$q = QName('example-uri', 'ppp:example-lname')">OK</xsl:if></t>;
    <f><xsl:if test="$q = QName('example-uri2', 'example-lname')">ERROR</xsl:if></f>;   
    <f><xsl:if test="$q = QName('example-uri', 'qqq:example-lname2')">ERROR</xsl:if></f>;        
  </out>     
  </xsl:template>                             


</xsl:stylesheet>