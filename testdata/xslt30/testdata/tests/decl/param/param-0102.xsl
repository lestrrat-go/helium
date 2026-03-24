<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case that uses xsl:call-template instruction to set a parameter value to a 
                boolean, and uses it in an xsl:if instruction. -->

	<t:template match="doc">
		<out>
			<t:call-template name="foo">
				<t:with-param name="partest" select="contains('foo','o')"/>
			</t:call-template>
		</out>
	</t:template>

	<t:template name="foo">
		<t:param name="partest"/>
		<t:if test="$partest">
			<t:text>Success</t:text>
		</t:if>
	</t:template>
</t:transform>
