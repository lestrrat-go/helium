<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: A complex pattern that has to be evaluated "the hard way", including the 'child-or-top' logic -->
  <!-- Note that rewriting A/(B except C) as (A/B except A/C) is in general incorrect. In this particular
       instance, //x/(descendant::a except child::a) delivers <a>25</a>, while 
       //x/descendant::a except //x/child::a delivers an empty sequence.-->

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

  <xsl:template match="x/(descendant::a except child::a)" priority="20">
    <AA>
      <xsl:apply-templates/>
    </AA>
  </xsl:template>
  
  <xsl:template match="a" priority="10">
    <aa>
      <xsl:apply-templates/>
    </aa>
  </xsl:template>



</xsl:transform>
