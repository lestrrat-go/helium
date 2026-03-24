<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xxx="http://www.w3.org/1999/XSL/Transform"
  version="2.0">
  <xsl:namespace-alias result-prefix="xsl" stylesheet-prefix="xxx"/>
  
  <xsl:template match="/bar">
    <xsl:variable name="bar" as="element(bar)">
    <bar>
      <xsl:copy-of select="@*"/>
      <xsl:copy-of select="node()"/>
    </bar> 
    </xsl:variable>
    <xsl:apply-templates select="$bar"/>
  </xsl:template>
  <xsl:template match="/foo">
    <xxx:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
          <xsl:apply-templates select="*"/>
    </xxx:stylesheet>
  </xsl:template>
</xsl:stylesheet>