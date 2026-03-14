<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: An "except" pattern using the child axis -->

  <xsl:mode on-no-match="shallow-copy"/>

  <xsl:variable name="nodes" as="element()*">
    <x>
      <p>1</p>
      <q>2</q>
      <r>3</r>
      <s>4</s>
      <my:z xmlns:my="http://my.com/">5</my:z>
    </x>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
    <x>
      <xsl:apply-templates select="$nodes/*"/>
    </x>
  </xsl:template>

  <xsl:template match="* except q except Q{http://my.com/}z" priority="20">
    <xsl:element name="{local-name()}">{.+1}</xsl:element>
  </xsl:template>
  
  <xsl:template match="* except (xxx | yyy)" priority="10"/>
 



</xsl:transform>
