<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Purpose: Test next-match with two imported stylesheets of different precedence-->
  
  <xsl:import href="next-match-006a.xsl" />
  
  <xsl:template match="node()">
    <a>
      <xsl:next-match />
    </a>
  </xsl:template>
  
  <xsl:template match="doc">
    <b>
      <xsl:next-match />
    </b>
  </xsl:template>
  
  <xsl:template match="*[foo]">
    <c>
      <xsl:next-match />
    </c>
  </xsl:template>
  
  
  <!-- Copyright IBM Corp. 2004, 2005. -->
  
</xsl:stylesheet>
