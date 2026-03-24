<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that zero-length text nodes are removed when 
  				they appear in a sequence that is used to form the content of a node.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:text>Level 1</t:text>
         <t:text/>
         <t:text>Level 2</t:text>
         <t:text/>
         <t:text>level 3</t:text>
      </out>
	  </t:template>
</t:transform>
