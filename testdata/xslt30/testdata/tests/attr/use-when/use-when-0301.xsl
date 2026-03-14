<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case that uses use-attribute-set in an xsl:element instruction,and the 
  				attribute-set has a use-when attribute which evaluates to true. -->

	<t:template match="/">
		<out>
			<t:apply-templates/>
		</out>
	</t:template>

	<t:template match="elem1">
		<t:copy use-attribute-sets="mySet">
			<t:attribute name="attB" select="@value"/>
		</t:copy>
	</t:template>

	<t:template match="elem2">
		<t:element name="e2">
			<t:attribute name="attB">Explicit value</t:attribute>
			<t:element name="e3" use-attribute-sets="mySet"/>
		</t:element>
	</t:template>

	<t:attribute-set name="mySet" use-when="2-1=1">
		<t:attribute name="attB" select="/doc/data/elem2/@value"/>
	</t:attribute-set>
</t:transform>
