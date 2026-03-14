<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Test case to verify that creation of multiple elements at the top level 
  				(directly under the document node) is not an error-->

	<t:template match="doc">
		<t:document>
			<t:copy-of select="foo"/>
		</t:document>
	</t:template>
</t:transform>
