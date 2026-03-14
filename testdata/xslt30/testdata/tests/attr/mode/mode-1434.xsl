<?xml version="1.0"?>

<!-- mode-default002 -->
<!-- Michael Kay -->
<!-- default-mode = the unnamed mode --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  default-mode="#unnamed">
 
  <xsl:template match="/">
    <root><xsl:apply-templates mode="#unnamed"/></root>
  </xsl:template>
  
  <xsl:template match="bktshort">
    <book>
      <xsl:apply-templates mode="#current"/>
    </book>
  </xsl:template>
  
  <xsl:template match="bktshort/text()" mode="#unnamed">
    <QED><xsl:value-of select="."/></QED>
  </xsl:template>
  
  <xsl:template match="v | chapter/text()"/>
   
</xsl:stylesheet>