<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of pattern matching of parentless attribute node with attribute axis 
       and the Node Test in Pattern Step is the NameTest "*" -->

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

<xsl:variable name="att" as="attribute()">
	<xsl:attribute name="my_att">att-value</xsl:attribute>
</xsl:variable>

<xsl:template match="doc">
	<out>
		<xsl:apply-templates select="$elem"/>
		<xsl:apply-templates select="$pi1"/>
		<xsl:apply-templates select="$comm"/>
	  	<xsl:apply-templates select="$txt"/>
	  	<xsl:apply-templates select="$att"/>
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

<xsl:template match="attribute::*">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

</xsl:stylesheet>
