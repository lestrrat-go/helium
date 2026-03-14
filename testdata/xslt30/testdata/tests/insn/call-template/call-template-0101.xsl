<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Launch transformation with initial template specified by NCName. -->

	<t:template match="/">
		<t:comment>ERROR: this value should NOT appear</t:comment>
	</t:template>

	<t:template name="temp">
		<out>
			<t:text>Do Nothing!</t:text>
		</out>
	</t:template>
</t:transform>
