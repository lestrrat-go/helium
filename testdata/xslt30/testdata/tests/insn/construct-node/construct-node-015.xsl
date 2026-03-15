<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:processing-instruction to create two processing-instruction nodes.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:processing-instruction name="pi_1" select="'Inside select attribute'"/>
         <t:processing-instruction name="pi_2">Inside the sequence constructor</t:processing-instruction>
      </out>
	  </t:template>
</t:transform>
