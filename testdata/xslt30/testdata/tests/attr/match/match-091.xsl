<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Purpose: Test of patterns with name test  *:NCName -->

<xsl:template match="doc">
	<out><xsl:apply-templates/></out>
</xsl:template>

<xsl:template match="text()"/>

<xsl:template match="document-node()/child::*:doc/*:inner1">
	<t1>
		<xsl:value-of select="name(.)"/>
	</t1>
</xsl:template>

<xsl:template match="//*:inner2">
	<t2>
		<xsl:value-of select="name(.)"/>
	</t2>
</xsl:template>



</xsl:stylesheet>
