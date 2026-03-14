<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that the leading whitespace from the string value of a PI are striped.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:text>
</t:text>
         <t:processing-instruction name="pi_1" select="'   Inside select attribute.'"/>
         <t:text>
</t:text>
         <t:processing-instruction name="pi_2">   Inside the sequence constructor.</t:processing-instruction>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
