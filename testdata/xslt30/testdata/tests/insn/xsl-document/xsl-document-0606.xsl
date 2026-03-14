<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests using xsl:use-when instruction inside an xsl:document.  -->

	<t:variable name="v" as="document-node()">
		<t:document>
			<t:element name="e1" use-when="true()">
				<t:attribute name="attr1" select="'value1'" use-when="1=1"/>
			</t:element>
			<e2 t:use-when="0=-0"/>
			<e3 t:use-when="false()">false</e3>
			<t:element name="e4" use-when="false()">
				<t:attribute name="attr1" select="'value1'" use-when="1=1"/>
			</t:element>
		</t:document>
	</t:variable>

	<t:template match="/">
		<out>
			<t:copy-of select="$v"/>
		</out>
	</t:template>

	<t:template match="text()"/>
</t:transform>
