<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:t="http://www.w3.org/1999/XSL/Transform"
	exclude-result-prefixes="foo" version="2.0">
	<!-- Purpose: Test the "required" attribute for xsl:with-param.
                This MUST generate an error, as the required attribute is not permitted.  -->

	<t:function name="foo:func">
		<t:call-template name="temp1">
			<t:with-param name="par" select="'xyz'" required="yes"/>
		</t:call-template>
	</t:function>

	<t:template match="/">
		<t:apply-templates select="doc">
			<t:with-param name="par" select="'abc'" required="yes"/>
		</t:apply-templates>
	</t:template>

	<t:template match="doc">
		<out>
			<t:value-of select="foo:func()"/>
		</out>
	</t:template>

	<t:template name="temp1">
		<t:param name="par" select="'123'" required="yes"/>
		<t:value-of select="$par"/>
	</t:template>
</t:transform>
