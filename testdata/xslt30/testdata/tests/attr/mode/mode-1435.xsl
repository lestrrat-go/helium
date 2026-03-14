<?xml version="1.0"?>

<!-- mode-default003 -->
<!-- Michael Kay -->
<!-- default-mode = a namespaced mode --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs a"
  default-mode=" a:mode " xmlns:a="http://example.com/a">
 
  <xsl:template match="/">
    <root><xsl:apply-templates/></root>
  </xsl:template>
  
  <xsl:template match="bktshort">
    <book>
      <xsl:apply-templates mode="#current"/>
    </book>
  </xsl:template>
  
  <xsl:template match="bktshort/text()" mode="b:mode" xmlns:b="http://example.com/a" exclude-result-prefixes="b">
    <QED><xsl:value-of select="."/></QED>
  </xsl:template>
  
  <xsl:template match="v | chapter/text()"/>
   
</xsl:stylesheet>