<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of patterns with kind test attribute() -->

<xsl:template match="/">
	<out><xsl:apply-templates select="//@*"/></out>
</xsl:template>

<xsl:template match="document-node()/child::*/child::*/attribute()">
	<second>
		<xsl:value-of select="name(.)"/>
	</second>
</xsl:template>

<xsl:template match="document-node()/doc/attribute()">
	<doc>
		<xsl:value-of select="name(.)"/>
	</doc>
</xsl:template>

</xsl:stylesheet>
