<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Simple test with a tunnel param specified within xsl:call-template -->

	<t:template match="/">
		<out>
			<t:call-template name="temp">
				<t:with-param name="par1" select="'foo'" tunnel="yes"/>
			</t:call-template>
		</out>
	</t:template>

	<t:template name="temp">
		<t:param name="par1" select="'xyz'" tunnel="yes"/>
		<t:value-of select="$par1"/>
	</t:template>
</t:transform>
