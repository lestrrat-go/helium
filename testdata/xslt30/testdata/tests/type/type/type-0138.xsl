<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test use of element(*) in a path expression. -->
  <?spec xpath#id-element-test?>
  
 
  <xsl:template match="/">
  <out>
    <xsl:for-each select="//element(*)">
       <xsl:value-of select="name()"/>
       <xsl:text>, </xsl:text>
    </xsl:for-each>
  </out>     
  </xsl:template>                             


</xsl:stylesheet>