<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case where the intial template launches on an element other than root. 
  				Initially the template gets invoked and does not have any tunnel parameters.
  				Further processing would invoke another template which is aimed to be called
  				on the same initial node and passes a tunnel parameters to it. -->
	
	<t:template match="/">
		<doc><t:apply-templates/></doc>
	</t:template>

	<t:template match="data">
		<t:param name="foo" select="'default_value'" tunnel="yes"/>
		<out>
			<t:text>Parameter foo is: </t:text>
			<t:value-of select="$foo"/>
		</out>
		<t:if test="($foo='default_value')">
			<t:apply-templates/>
		</t:if>
	</t:template>

	<t:template match="inner">
		<t:call-template name="A"/>
	</t:template>

	<t:template name="A">
		<t:apply-templates select="//data">
			<t:with-param name="foo" select="'passed_value'" tunnel="yes"/>
		</t:apply-templates>
	</t:template>

	<t:template match="text()"/>
</t:transform>
