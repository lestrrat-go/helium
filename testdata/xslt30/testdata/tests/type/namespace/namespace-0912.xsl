<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:w="w-uri" xmlns:x="x-uri"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  
  <xsl:mode on-multiple-match="fail" on-no-match="shallow-copy" />
  
  <xsl:template as="element(x:report)" match="/">
    
    <xsl:variable as="element(p)" name="var">
      <xsl:apply-templates />
    </xsl:variable>
    
    <xsl:element name="report" namespace="x-uri">
      <xsl:sequence select="$var" />
    </xsl:element>
  </xsl:template>
  
  <xsl:template as="element(w:s)" match="q">
    <xsl:element name="w:s" namespace="w-uri" />
  </xsl:template>
  
</xsl:stylesheet>