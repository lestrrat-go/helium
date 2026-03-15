<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
     xmlns:a="http://a.uri/" xmlns:b="http://b.uri/">
     
     <?spec xpath#node-tests?>

  <xsl:template match="/">
    <o><xsl:apply-templates select="//@*"/></o>
  </xsl:template>
  
  <xsl:template match="@a:*">[<xsl:value-of select="name()"/>]</xsl:template>
  <xsl:template match="@b:*">[]</xsl:template>  
 
</xsl:stylesheet>
