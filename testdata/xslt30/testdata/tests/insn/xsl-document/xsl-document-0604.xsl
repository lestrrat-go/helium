<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests creating an element inside an xsl:document and use xsl:use-attribute-set to add attributes
  				to the new element inside the document.  -->

	<t:attribute-set name="attset">
		<t:attribute name="t1">1</t:attribute>
		<t:attribute name="t2" select="2"/>
		<t:attribute name="t3">
			<t:value-of select="3"/>
		</t:attribute>
	</t:attribute-set>
	
	<t:variable name="v" as="document-node()">
		<t:document>
			<a t:use-attribute-sets="attset"/>
		</t:document>
	</t:variable>

	<t:template match="doc">
		<out>
			<t:copy-of select="$v"/>
		</out>
	</t:template>

</t:transform>
