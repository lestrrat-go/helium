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
			<data>A.1</data>
			<data>B.1</data>
			<data>B.2</data>
		</doc>
	</xsl:variable>

	<xsl:key name="data1-by-init" match="data" use="replace(.,'\..$','')"/>

	<xsl:key name="data2-by-init" match="ex:data" use="replace(.,'\..$','')" xmlns:ex="example.com"/>

	<xsl:template name="xsl:initial-template">
		<xsl:apply-templates select="$data/*"/>
	</xsl:template>
	
	<xsl:template match="doc">
		<report>
			<xsl:apply-templates/>
		</report>
    </xsl:template>

	<xsl:template match="data" expand-text="true">
		<run>
			<data1>data1 key gives me { count(key('data1-by-init',replace(.,'\..$',''))) }</data1>
			<data2>data2 key gives me { count(key('data2-by-init',replace(.,'\..$',''))) }</data2>
		</run>
	</xsl:template>
    
</xsl:stylesheet>