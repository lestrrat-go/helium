<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests the tunnel parameters specified within xsl:apply-imports 
       with both tunnel="no" and tunnel="yes" attributes. -->

	<t:import href="import2.xsl"/>

	<t:template match="doc">
		<out>
			<t:apply-imports>
				<t:with-param name="par" select="'foo'" tunnel="no"/>
				<t:with-param name="par2" select="'bar'" tunnel="yes"/>
			</t:apply-imports>
		</out>
	</t:template>

	<t:template match="data">
		<t:param name="par" select="'xyz'" tunnel="yes"/>
		<t:param name="par2" select="'zyx'" tunnel="yes"/>
		<t:value-of select="$par"/> and <t:value-of select="$par2"/>
	</t:template>

	<t:template match="text()"/>
</t:transform>
