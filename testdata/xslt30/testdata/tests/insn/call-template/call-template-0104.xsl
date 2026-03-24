<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Launched with initial template but there is no template of that name, where the name is an NCName. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <out/>
   </t:template>

   <t:template name="temp">
	     <t:text> Do Nothing! </t:text>
   </t:template>

   <t:template xmlns:my="http://www.othertemp.com" name="my:temp">
      <t:text> Do Something! </t:text>
   </t:template>
</t:transform>
