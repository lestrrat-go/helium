<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of pattern matching of parentless attribute node -->

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

<xsl:variable name="att1" as="attribute()">
  <xsl:attribute name="my_att1">att1-value</xsl:attribute>
</xsl:variable>

<xsl:variable name="att2" as="attribute()">
  <xsl:attribute name="my_att2">att2-value</xsl:attribute>
</xsl:variable>

<xsl:template match="doc">
  <out>
	<xsl:apply-templates select="$elem"/>
	<xsl:apply-templates select="$pi1"/>
	<xsl:apply-templates select="$comm"/>
 	<xsl:apply-templates select="$txt"/>
  	<xsl:apply-templates select="$att1"/>
  	<xsl:apply-templates select="$att2"/>
  </out>
</xsl:template>

<xsl:template match="attribute::element()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::processing-instruction()" priority="2">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::text()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::comment()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::*:my_att1">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::my_att2">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

</xsl:stylesheet>
