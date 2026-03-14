<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- Test of pattern with range expression inside predicate -->

<xsl:template match="@*" priority="1"/>

<xsl:template match="processing-instruction()"/>

<xsl:template match="element()"/>

<xsl:template match="text()"/>

<xsl:template match="comment()"/>

<xsl:template match="doc">
	<xsl:apply-templates select="//attribute::attribute()"/>
</xsl:template>

<xsl:template match="document-node()/child::*/attribute()[1 to 1]" priority="2">
<t>
	<xsl:value-of select="name(.)"/>
</t>
</xsl:template>

</xsl:stylesheet>
