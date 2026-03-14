<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="2.0"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:template match="e[@a1]"/>
	<xsl:template match="e[@a2]"/>
	<xsl:template match="e[@a3]"/>
	<xsl:template match="e[@a4]"/>
	<xsl:template match="e[@a5]"/>
	<xsl:template match="e[@a6]"/>
	<xsl:template match="e[@a7]"/>
	<xsl:template match="e[@a]">found</xsl:template>
	<xsl:template match="e[@a = 'a']"/>
	<xsl:template match="e[@a = 'b']"/>
	<xsl:template match="e[@a = 'c']"/>
	<xsl:template match="e[@a = 'd']"/>
	<xsl:template match="e[@a = 'e']"/>
	<xsl:template match="e[@a = 'f']"/>
	<xsl:template match="e[@a = 'g']"/>
	<xsl:template match="e[@a = 'h']"/>
	<xsl:template match="e[@a = 'i']"/>
	<xsl:template match="e[@a = 'j']"/>
	<xsl:template match="e[@a = 'k']"/>
	<xsl:template match="e[@a = 'l']"/>
	<xsl:template match="e[@a = 'm']"/>
	<xsl:template match="e[@a = 'n']"/>
	<xsl:template match="e[@a = 'o']"/>
	<xsl:template match="e[@a = 'p']"/>
	<xsl:template match="e[@a = 'q']"/>
	<xsl:template match="e[@a = 'r']"/>
	<xsl:template match="e[@a = 's']"/>
	<xsl:template match="e[@a = 't']"/>
	<xsl:template match="e[@a = 'u']"/>
	<xsl:template match="e[@a = 'w']"/>
	<xsl:template match="e[@a = 'x']"/>
	<xsl:template match="e[@a = 'y']"/>
	<xsl:template match="e[@a = 'z']">
		<out><xsl:next-match/></out>
	</xsl:template>
	
	<xsl:template name="xsl:initial-template">
		<xsl:variable name="in" as="element()">
			<e a='z'/>
		</xsl:variable>
		<xsl:apply-templates select="$in"/>
	</xsl:template>
</xsl:stylesheet>
