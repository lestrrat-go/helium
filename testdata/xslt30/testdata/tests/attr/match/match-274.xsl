<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: matching parentless element using 'child-or-top' logic -->

  <xsl:mode on-no-match="shallow-copy"/>

  <xsl:variable name="nodes" as="element()*">
    <x>
      <a>23</a>
      <x>
        <a>25</a>
      </x>
    </x>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
    <xsl:apply-templates select="$nodes"/>
  </xsl:template>

  <xsl:template match="x/descendant-or-self::x[position() = year-from-date(current-date()) idiv 2000]" priority="20">
    <XX>
      <xsl:apply-templates/>
    </XX>
  </xsl:template>
  
  <xsl:template match="x" priority="10">
    <xx>
      <xsl:apply-templates/>
    </xx>
  </xsl:template>



</xsl:transform>
