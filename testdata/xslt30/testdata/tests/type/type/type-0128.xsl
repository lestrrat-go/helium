<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test QName construction and extraction. -->
  <?spec fo#func-QName?>
  <?spec fo#func-local-name-from-QName?>
  <?spec fo#func-namespace-uri-from-QName?>  
  
  <xsl:param name="p"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="q" as="xs:QName" select="QName('example-uri', 'example-lname')"/>
    Local Name: <xsl:value-of select="local-name-from-QName($q)"/>;
    URI: <xsl:value-of select="namespace-uri-from-QName($q)"/>;    
    <xsl:variable name="q2" as="xs:QName" select="QName((), 'example-lname')"/>
    Local Name: <xsl:value-of select="local-name-from-QName($q2)"/>;
    URI: <xsl:value-of select="namespace-uri-from-QName($q2)"/>;  
  </out>     
  </xsl:template>                             


</xsl:stylesheet>