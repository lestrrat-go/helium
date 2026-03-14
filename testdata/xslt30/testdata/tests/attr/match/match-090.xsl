<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Purpose: Test of patterns with kind test element() -->

<xsl:template match="doc">
	<out><xsl:apply-templates/></out>
</xsl:template>

<xsl:template match="text()"/>

<xsl:template match="/child::*/element()">
	<t1>
		<xsl:value-of select="name(.)"/>
	</t1>
	<xsl:apply-templates/>
</xsl:template>

<xsl:template match="element()/child::*:inner1/element()">
	<t2>
		<xsl:value-of select="name(.)"/>
	</t2>
</xsl:template>

</xsl:stylesheet>
