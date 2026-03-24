<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests xsl:choose instruction inside an xsl:document.  -->

	<t:variable name="v" as="document-node()">
		<t:document>
			<t:choose>
				<t:when test="1=1">
					<t:element name="a">
						<t:attribute name="status" select="boolean('a')"/>
					</t:element>
				</t:when>
				<t:otherwise>
					<b>false</b>
				</t:otherwise>
			</t:choose>
		</t:document>
	</t:variable>

	<t:template match="doc">
		<out>
			<t:copy-of select="$v"/>
		</out>
	</t:template>
</t:transform>
