<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test the tunnel parameter specified within xsl:apply-imports and 
  	   the ttribute tunnel="yes" to make sure the parameter is passed on by the 
  	   called template to any further template that it calls. -->

	<t:import href="import1.xsl"/>

	<t:template match="doc">
		<out>
			<t:apply-imports>
				<t:with-param name="par" select="'abc'" tunnel="yes"/>
			</t:apply-imports>
		</out>
	</t:template>

	<t:template match="data">
		<t:param name="par" select="'123'" tunnel="yes"/>
		<t:value-of select="$par"/>
	</t:template>

	<t:template match="text()"/>
</t:transform>
