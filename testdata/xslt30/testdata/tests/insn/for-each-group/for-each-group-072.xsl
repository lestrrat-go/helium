<?xml version="1.0" encoding="UTF-8" ?>

<xsl:stylesheet version="3.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
	xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	exclude-result-prefixes="xs">

<xsl:variable name="doc">
	<a>
		<b color="blue">
			<object quantity="3"/>
			<object quantity="1"/>
			<object quantity="2"/>
			<object quantity="3"/>
			<object quantity="3"/>
		</b>
		<b color="red">
			<object quantity="1"/>
			<object quantity="2"/>
			<object quantity="3"/>
			<object quantity="3"/>
		</b>
	</a>
</xsl:variable>

	<xsl:template name="main">
	<out>
		<xsl:for-each-group select="$doc//object" group-by="@quantity,../@color" composite="yes">
			<group key="{current-grouping-key()}" key.count="{count(current-grouping-key())}" group.size="{count(current-group())}"/>
		</xsl:for-each-group>
	</out>	
	</xsl:template>

</xsl:stylesheet>
