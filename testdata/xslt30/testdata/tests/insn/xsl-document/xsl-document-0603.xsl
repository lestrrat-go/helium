<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests using xsl:document instruction inside an xsl:message instruction. -->

	<t:variable name="v" as="document-node()">
		<t:document>
			<t:text>* Message one *</t:text>
		</t:document>
	</t:variable>

	<t:template match="doc">
		<out>
			<t:message select="$v">
				<t:document>
					<t:text>* Message two *</t:text>
				</t:document>
			</t:message>
		</out>
	</t:template>
</t:transform>
