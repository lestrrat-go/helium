<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case to verify that a document node can contain any sequence of elements, 
  				text nodes, comments, and processing instructions as its children.-->

	<t:template match="root">
		<t:param name="pi" select="child[2]"/>
		<out>
			<t:document>
				<t:variable name="comment" select="'Valuable comment'"/>
				<t:comment select="$comment"/>
				<t:element name="new">
					<t:attribute name="att1">fresh</t:attribute>
				</t:element>
				<lre/>
				<t:processing-instruction name="my_pi" select="$pi"/>
				<t:document>
					<my:child xmlns:my="http://www.examples.com/my" />
				</t:document>
				<t:text>Text Node</t:text>
				<t:call-template name="createNode"/>
				<t:value-of select="child"/>
			</t:document>
		</out>
	</t:template>

	<t:template match="child"/>

	<t:template name="createNode">
		<newLRE>
			<t:element name="ch"/>
		</newLRE>
	</t:template>
</t:transform>
