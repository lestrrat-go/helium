<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case that uses use-when attribute on an element that has a static 
  				error, the path expression of the attribute evaluates to false and
  				therefore no error will be reported.  -->

	<t:template match="doc">
		<out>
			<t:element name="e">
				<t:element name="e1" use-when="1=2">
					<t:element name="e1.1">v1.1</t:element>
					<t:element name="e1.2">v1.2</t:element>
					<t:element name="e1.3">v1.3</t:element>
				</t:element>

				<t:element name="e2">
					<t:element name="e2.1">v2.1</t:element>
					<t:element name="e2.2" use-when="true()">v2.2</t:element>
					<t:element name="e2.3" use-when="false()">
						<t:attribute name="e2.3.attr1">attr1</t:attribute>
						<t:element name="e2.3.1">
							<t:value-of select="e2.3.1">e2.3.1</t:value-of>
						</t:element>
						<t:attribute name="e2.3.attr2">attr2</t:attribute>
						<t:element name="e2.3.2">e2.3.2</t:element>
					</t:element>
				</t:element>
			</t:element>
		</out>
	</t:template>
</t:transform>
