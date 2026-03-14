<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!--Test of pattern matching of parentless copied attribute node with attribute axis and the NodeTest in PatternStep is various kindTest -->

<xsl:template match="doc">
	<xsl:variable name="elem1" as="element()+">
		<xsl:copy-of select="element()"/>
	</xsl:variable>

	<xsl:variable name="pi1" as="processing-instruction()+">
		<xsl:copy-of select="processing-instruction()"/>
	</xsl:variable>

	<xsl:variable name="comm1" as="comment()">
		<xsl:copy-of select="comment()"/>
	</xsl:variable>

	<xsl:variable name="txt1" as="text()">
		<xsl:copy-of select="text()"/>
	</xsl:variable>

	<xsl:variable name="att1" as="attribute()">
		<xsl:copy-of select="attribute()"/>
	</xsl:variable>

	<xsl:apply-templates select="$elem1"/>
	<xsl:apply-templates select="$pi1"/>
	<xsl:apply-templates select="$comm1"/>
	<xsl:apply-templates select="$txt1"/>
  	<xsl:apply-templates select="$att1"/>
</xsl:template>

<xsl:template match="attribute::element()">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="attribute::processing-instruction()" >
  <t>
	<xsl:value-of select="string()"/>
  </t>
</xsl:template>

<xsl:template match="attribute::text()">
  <t>
	<xsl:value-of select="string()"/>
  </t>
</xsl:template>

<xsl:template match="attribute::comment()">
  <t>
	<xsl:value-of select="string()"/>
  </t>
</xsl:template>

<xsl:template match="attribute::attribute()">
  <t>
	<xsl:value-of select="string()"/>
  </t>
</xsl:template>

<xsl:template match="text()" priority="1"/>

<xsl:template match="comment()" priority="1"/>

<xsl:template match="processing-instruction()" priority="1"/>


</xsl:stylesheet>
