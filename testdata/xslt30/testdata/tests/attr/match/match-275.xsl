<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: the pattern A except B is counter-intuitive -->

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

  <xsl:template match="descendant::a except child::a" priority="20">
    <!-- Intuitively, you might think that this matches any node that matches descendant::a 
         but does not match child::a, which would match nothing. -->
    <!-- The actual semantics are that it matches any node N that has an ancestor $A such that N is selected
         by $A/descendant::a but not by $A/child::a - which means it matches any 'a' element that has
         a grandparent. (Complicating this further, child::a means child-or-top::a, but that doesn't
         affect this test.) -->
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
