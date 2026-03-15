<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: Same as match-282 but with a predicate on the first step that doesn't match -->


  <xsl:variable name="nodes" as="element()*">
    <x id="1">
      <a>23</a>
      <z>
        <b>25</b>
      </z>
    </x>
  </xsl:variable>
  
  <xsl:mode on-no-match="shallow-copy"/>


  <xsl:template name="xsl:initial-template">
    <xsl:apply-templates select="$nodes"/>
  </xsl:template>

  <xsl:template match="x[@id='2']/(child::a|descendant::b)">
    <wrong/>
  </xsl:template>


</xsl:transform>
