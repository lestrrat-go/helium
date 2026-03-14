<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/02/xpath-datatypes"
exclude-result-prefixes="xs xdt">

  <!-- PURPOSE: Test anyURI to string promotion. -->
  <?spec fo#casting-to-string?>

  
  <xsl:param name="p" select="xs:anyURI('http://www.example.com/example-uri')"
             as="xs:anyURI"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <i><xsl:value-of select="$p instance of xs:string"/></i>
    <c><xsl:value-of select="$p = 'http://www.example.com/example-uri'"/></c>
    <c><xsl:value-of select="$p eq 'http://www.example.com/example-uri'"/></c>
    <f><xsl:value-of select="starts-with($p, 'http://')"/></f> 
    <f><xsl:value-of select="string-length($p)"/></f>        
  </out>     
  </xsl:template>                             


</xsl:stylesheet>