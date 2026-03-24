<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: An "except" pattern of two global variables. -->
  
  <xsl:mode on-no-match="deep-skip"/>
  
  <xsl:variable name="nodes" as="element()*">
    <xsl:for-each select="1 to 100">
      <n>{.}</n>
    </xsl:for-each>
  </xsl:variable>
  
  <xsl:variable name="threes" select="$nodes[. mod 3 = 0]"/>
  <xsl:variable name="sevens" select="$nodes[. mod 7 = 0]"/>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:apply-templates select="$nodes"/>
    </out>
  </xsl:template>
  
  <xsl:template match="$sevens except $threes">
    <xsl:sequence select="."/>
  </xsl:template>
  
  
  
  
  
  
</xsl:transform>
