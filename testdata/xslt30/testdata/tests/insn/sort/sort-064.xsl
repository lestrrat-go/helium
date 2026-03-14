<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:Sort a sequence of attributes -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="list" as="attribute()*">
		    <t:attribute name="att1">
         <t:value-of select="'z'"/>
      </t:attribute>
		    <t:attribute name="att2">
         <t:value-of select="'a'"/>
      </t:attribute>
		    <t:attribute name="att3" select="'V3'"/>
		    <t:attribute name="att4" select="'V_4'"/>
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
