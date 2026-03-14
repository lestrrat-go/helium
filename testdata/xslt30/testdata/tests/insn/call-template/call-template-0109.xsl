<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Use EQName for template name. -->

	<t:template match="/">
		<t:comment>ERROR: this value should NOT appear</t:comment>
	</t:template>

	<t:template name=" Q{}temp ">
			<t:call-template name="Q{http://another/}temp"/>	
	</t:template>
	
	<t:template name=" Q{http://another/}temp ">
		<out>
			<t:text>Do Nothing!</t:text>
		</out>
	</t:template>
</t:transform>
