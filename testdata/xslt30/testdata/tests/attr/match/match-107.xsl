<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" >

  <!-- Purpose: Test of template matching of various parentless nodes, child axis is used explicitly and the NodeTest in PatternStep is various KindTest -->

<xsl:variable name="elem1" as="element()">
  <xsl:element name="a">elem_a_value</xsl:element>
</xsl:variable>

<xsl:variable name="pi1" as="processing-instruction()">
	<xsl:processing-instruction name="pi">PI_data</xsl:processing-instruction>
</xsl:variable>

<xsl:variable name="pi2" as="processing-instruction()">
	<xsl:processing-instruction name="anotherpi">another_PI_data</xsl:processing-instruction>
</xsl:variable>

<xsl:variable name="comm1" as="comment()">
  <xsl:comment>This is the 1st comment</xsl:comment>
</xsl:variable>

<xsl:variable name="txt1" as="text()">
  <xsl:text>text-in-doc</xsl:text>
</xsl:variable>

<xsl:template match="doc">
	<out>
		<xsl:apply-templates select="$elem1"/>
		<xsl:apply-templates select="$pi1"/>
		<xsl:apply-templates select="$pi2"/>
		<xsl:apply-templates select="$comm1"/>
	  	<xsl:apply-templates select="$txt1"/>
  	</out>
</xsl:template>

<xsl:template match="child::element()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::processing-instruction()" priority="2">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::processing-instruction(anotherpi)" priority="1">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::text()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::comment()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

</xsl:stylesheet>
