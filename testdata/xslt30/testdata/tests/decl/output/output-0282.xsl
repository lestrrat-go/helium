<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised on invalid @standalone value in xsl:output.-->

   <t:output method="xml" encoding="UTF-8" indent="no" standalone="true"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
