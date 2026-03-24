<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  version="3.0">
  
  
  <xsl:accumulator name="a1" as="xs:string" initial-value="'init'">
    <xsl:accumulator-rule match="foo" select="$value || ', matched ' || path()"/>
  </xsl:accumulator>
  
  <xsl:param name="p1">
    <foo/>
  </xsl:param>
  
  <xsl:param name="p2" as="element(foo)">
    <foo/>
  </xsl:param>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:apply-templates select="$p1/node(), $p2"/>
    </out>
  </xsl:template>
  
  <xsl:template match="foo">
    <processed value="{accumulator-before('a1')}" 
      root="{node-name(root())}"
      path="{path()}" instance-of-foo="{. instance of element(foo)}"/>
  </xsl:template>
  
</xsl:stylesheet>