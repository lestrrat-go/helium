<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0"
    xpath-default-namespace="example.com">

	<!-- Run on itself to demo a bug -->
	
	<xsl:variable name="data">
		<doc xmlns="example.com">
			<data><id>A1</id>A.1</data>
			<data><id>B1</id>B.1</data>
			<data><id>B2</id>B.2</data>
		</doc>
	</xsl:variable>

	<xsl:key name="data-by-id" match="data" use="id"/>

	
	<xsl:template name="xsl:initial-template">
		<xsl:apply-templates select="$data/*"/>
	</xsl:template>
	
	<xsl:template match="doc">
		<report count="{count(key('data-by-id', 'A1')) }"/>
    </xsl:template>

    
</xsl:stylesheet>