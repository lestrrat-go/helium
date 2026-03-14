<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that an static error is raised if a reserved namespace is used in the name of a named template. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <t:call-template name="t:temp"/>
   </t:template>

   <t:template name="t:temp">
	     <t:text> Do Nothing! </t:text>
   </t:template>
</t:transform>
