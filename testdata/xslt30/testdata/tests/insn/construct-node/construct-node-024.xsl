<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of instructions inside xsl:text -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:text>
            <t:if test="true()">Success</t:if>
         </t:text>
      </out>
	  </t:template>
</t:transform>
