<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that If the result of evaluating the content of the 
  				xsl:processing-instruction contains the string ?>, this string is 
  				modified by inserting a space between the ? and > characters.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:processing-instruction name="pi_1" select="'What if we have ?&gt;'"/>
         <t:processing-instruction name="pi_2">What if we have ?&gt;</t:processing-instruction>
      </out>
	  </t:template>
</t:transform>
