<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that in the generated comment node, the processor 
  				inserts a space after any occurrence of hypen that is followed by 
  				another hyphen or that ends the comment. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
	     <out>
         <t:comment> Testing the double --</t:comment>
         <t:text>
</t:text>
         <t:comment> Testing the single -</t:comment>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
