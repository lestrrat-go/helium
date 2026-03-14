<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case to verify that a dynamic error is raised if the result of 
  				evaluating an XPath expression is required to be a lexical QName and
  				the defining element has no namespace node whose name matches the prefix 
  				of the lexical QName. -->
	
	<t:template name="main">
		<t:variable name="v" select="'my:name'"/>
		<out>
			<t:element name="a">
				<t:attribute name="{$v}"/>
			</t:element>
		</out>
	</t:template>
</t:transform>
