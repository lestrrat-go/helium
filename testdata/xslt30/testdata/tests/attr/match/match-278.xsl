<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: An "intersect" pattern using the descendant axis. The meaning here
       is somewhat counter-intuitive. -->

  <xsl:mode on-no-match="shallow-copy"/>

  <xsl:variable name="nodes">
    <div id="a">
      <div id="b">
        <div id="c">text</div>
      </div>
    </div>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
    <x>
      <xsl:apply-templates select="$nodes"/>
    </x>
  </xsl:template>

  <xsl:template match="div[@id='a']//* intersect div[@id='b']//*" priority="20">
    <!-- This (surprisingly) shouldn't match anything. The pattern is defined to match a node $N if
         root($N)/descendant-or-self::node()/(child::div[@id='a']//* intersect child::div[@id='b']//*) includes N. This
         can only be the case if the two child::div elements are evaluated against the same parent element,
         not if they are nested div elements -->
    <wrong/>
  </xsl:template>
  
 
 



</xsl:transform>
