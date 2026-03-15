<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that adjacent strings/text nodes that are copied 
  				into one string become the content of the xsl:message. -->

   <t:template match="docs">
   		<out>
         <t:message terminate="no" select="'Error Message:'">
   				     <t:text>T</t:text>
   				     <t:text>h</t:text>
   				     <t:text>i</t:text>
   				     <t:text>s</t:text>
   				     <t:text> </t:text>
	   			     <t:text>i</t:text>
   				     <t:text>s</t:text>
   				     <t:text> </t:text>
   				     <t:text>a</t:text>
   				     <t:text>n</t:text>
   				     <t:text> </t:text>
   				     <t:text/>
   				     <t:text>e</t:text>
   				     <t:text>r</t:text>
   				     <t:text>r</t:text>
   				     <t:text>o</t:text>
   				     <t:text>r</t:text>
   			   </t:message>
      </out>
	  </t:template>
</t:transform>
