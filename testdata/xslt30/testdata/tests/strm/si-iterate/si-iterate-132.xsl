<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:gml="http://www.opengis.net/gml"
	exclude-result-prefixes="xs gml"
	version="3.0">
	
	<xsl:mode streamable="yes"/>
	<xsl:output method="xml" indent="no"/>
	
	<xsl:template match="/">
		<result>
		    <xsl:iterate select="outermost(.//gml:posList)">
		        <xsl:param name="polygons" select="0" as="xs:integer"/>
		        <xsl:param name="points" select="0" as="xs:integer"/>
		        <xsl:on-completion>
		            <polygons><xsl:value-of select="$polygons"/></polygons>
		            <points><xsl:value-of select="$points"/></points>
		        </xsl:on-completion>
		        <xsl:next-iteration>
		            <xsl:with-param name="polygons" select="$polygons+1"/>
		            <xsl:with-param name="points" select="$points + count(tokenize(., '\s+')) idiv 3"/>
		        </xsl:next-iteration>
		    </xsl:iterate>
		</result>
	</xsl:template>
		
</xsl:stylesheet>
