<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:Sort a sequence of text nodes -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="list" as="text()*">
		    <t:text/>
		    <t:text>a</t:text>
		    <t:text>a</t:text>
		    <t:text>d</t:text>
		    <t:text>x</t:text>
		    <t:text> </t:text>
		    <t:text>s</t:text>
		    <t:text> </t:text>
	  </t:variable>

   <t:template match="/">
	     <out>
         <t:call-template name="temp"/>
      </out>
   </t:template>

   <t:template name="temp">
      <t:perform-sort select="$list">
         <t:sort select="."/>
      </t:perform-sort>
   </t:template>

   <t:template match="text()"/>
</t:transform>
