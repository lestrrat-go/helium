<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


  <!-- Purpose: Use same-named variables in inner and outer scopes, where inner is a for-each loop. -->
  
  <!-- MHK: legal in 2.0, illegal in 1.0 -->
 

<xsl:template match="/">
  <xsl:variable name="bar">outer</xsl:variable>
  <outer bar="{$bar}">
    <xsl:for-each select="./*">
      <xsl:variable name="bar">inner</xsl:variable>
      <inner bar="{$bar}"/>
    </xsl:for-each>
  </outer>
</xsl:template>
                
</xsl:stylesheet>