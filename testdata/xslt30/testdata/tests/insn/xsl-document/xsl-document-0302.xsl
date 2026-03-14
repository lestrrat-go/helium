<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that Adjacent text nodes within the sequence are combined 
  				into a single text node, without any space separator. Zero-length text nodes 
  				are removed completely.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
		    <t:document>
		       <out>
            <t:text>new</t:text>
            <t:text>ly</t:text>
            <t:text>born</t:text>
            <t:text/>
            <t:text/>
            <t:text>in</t:text>
            <t:text> </t:text>
            <t:text>XSLT</t:text>
         </out>
  		  </t:document>
	  </t:template>
</t:transform>
