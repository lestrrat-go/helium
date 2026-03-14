<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:apply-templates inside an xsl:comment instruction 
  				to convert the source tree content into an output comment. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
		    <out>
         <t:comment>
    			     <t:apply-templates/>
    		   </t:comment>
      </out>
	  </t:template>
</t:transform>
