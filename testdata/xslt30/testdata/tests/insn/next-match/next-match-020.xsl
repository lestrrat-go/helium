<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:next-match inside xsl:if.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
      <out>
         <t:if test="name(*)='one'">
    	       <t:next-match/>
         </t:if>
         <t:if test="name(*)='two'">
    	       <t:next-match/>
         </t:if>
      </out>
   </t:template>

   <t:template match="element()">
	     <one xmlns:xsl="http://www.w3.org/1999/XSL/Transform">one time</one>
   </t:template>
</t:transform>
