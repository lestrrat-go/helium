<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


  <!-- Test of pattern matching of various parentless nodes -->
  <!-- child axis is implicitly used and the NodeTest in PatternStep is KindTest-->

<xsl:variable name="elem" as="element()">
  <xsl:element name="a">elem_a_value</xsl:element>
</xsl:variable>

<xsl:variable name="pi1" as="processing-instruction()">
  <xsl:processing-instruction name="pi">PI_data</xsl:processing-instruction>
</xsl:variable>

<xsl:variable name="pi2" as="processing-instruction()">
  <xsl:processing-instruction name="anotherpi">another_PI_data</xsl:processing-instruction>
</xsl:variable>

<xsl:variable name="comm" as="comment()">
  <xsl:comment>This is the 1st comment</xsl:comment>
</xsl:variable>

<xsl:variable name="txt" as="text()">
  <xsl:text>text-in-doc</xsl:text>
</xsl:variable>

<xsl:template match="doc">
  <out>
	<xsl:apply-templates select="$elem"/>
	<xsl:apply-templates select="$pi1"/>
	<xsl:apply-templates select="$pi2"/>
	<xsl:apply-templates select="$comm"/>
  	<xsl:apply-templates select="$txt"/>
  </out>
</xsl:template>

<xsl:template match="element()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="processing-instruction()" priority="2">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="processing-instruction(anotherpi)" priority="1">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="comment()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

</xsl:stylesheet>
