<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case that creates an element using xsl:element instruction and uses 
  				different ways (xsl:attribute instruction, xsl:call-template, xsl:copy-of)
  				to add attributes to the newly created element.-->

	<t:template match="file">
		<out>
			<t:element name="e1">
				<t:attribute name="a1">
					<t:value-of select="1"/>
				</t:attribute>
				<t:call-template name="makeAttr"/>
				<t:copy-of select="@id"/>
			</t:element>
		</out>
	</t:template>

	<t:template name="makeAttr">
		<t:attribute name="a2" select="2"/>
	</t:template>
</t:transform>
