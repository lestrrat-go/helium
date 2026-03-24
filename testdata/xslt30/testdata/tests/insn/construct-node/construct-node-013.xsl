<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that separators are never inserted between adjacent text nodes.  -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:value-of separator="*">
 				       <t:sequence select="1 to 4"/>
 				       <t:text>a</t:text>
 				       <t:text>ab</t:text>
 				       <t:text>abc</t:text>
 				       <t:text>abcd</t:text>
 				       <t:sequence select="5 to 7"/> 
 			     </t:value-of>
      </out>
	  </t:template>
</t:transform>
