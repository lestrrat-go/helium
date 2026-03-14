<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: An "except" pattern using the attribute axis -->

  <xsl:mode on-no-match="shallow-copy"/>

  <xsl:variable name="nodes" as="element()*">
    <x>
      <a p="1" q="2" r="3" xml:id="id23">23</a>
    </x>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
    <xsl:apply-templates select="$nodes"/>
  </xsl:template>

  <xsl:template match="@* except @q except @xml:id" priority="20">
    <xsl:attribute name="{local-name()}" select=". + 1"/>
  </xsl:template>
  
  <xsl:template match="@* except (@xxx | @yyy)" priority="10"/>
 



</xsl:transform>
