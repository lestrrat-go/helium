<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:bar="http://www.bar.com"
	xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="foo bar" version="2.0">
	<!-- Purpose: test tunnel params with same local name but different namespace
		specified within xsl:call-template -->

	<t:template match="/">
		<out>
			<t:call-template name="temp1">
				<t:with-param name="foo:par1" select="'foo1'" tunnel="yes"/>
				<t:with-param name="bar:par1" select="'bar1'" tunnel="yes"/>
			</t:call-template>
		</out>
	</t:template>

	<t:template name="temp1">
		<t:call-template name="temp2"/>
	</t:template>

	<t:template name="temp2">
		<t:param name="bar:par1" select="'bar2'" tunnel="yes"/>
		<t:param name="foo:par1" select="'foo2'" tunnel="yes"/>
		<t:value-of select="$foo:par1"/> and <t:value-of select="$bar:par1"/>
	</t:template>
</t:transform>
