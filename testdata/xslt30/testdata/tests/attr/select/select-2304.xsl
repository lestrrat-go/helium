<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that in an xsl:value-of instruction adjacent text 
  				nodes in the sequence are merged into a single text node. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="v">
		    <t:text>Y</t:text>
		    <t:text>o</t:text>
		    <t:text>u</t:text>
		    <t:text>n</t:text>
		    <t:text>g</t:text>
		    <t:text>s</t:text>
		    <t:text>t</t:text>
		    <t:text> </t:text>
		    <t:text>k</t:text>
		    <t:text>i</t:text>
		    <t:text>d</t:text>
		    <t:text/>
	  </t:variable>

   <t:template match="doc">
		    <t:text>
</t:text>
   		<out>
         <t:text>
</t:text>
         <a>
            <t:value-of>
   					       <t:text>Old</t:text>
   					       <t:text>est </t:text>
   					       <t:text>History</t:text>
   				     </t:value-of>
         </a>
         <t:text>
</t:text>
         <b>
            <t:value-of>
   					       <t:sequence select="$v"/>
				        </t:value-of>
         </b>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
