<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to show that if there is no select attribute and no sequence constructor for an 
  				xsl:perform-sort instruction then the initial sequence (and therefore, the sorted sequence) 
  				is an empty sequence. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <out>
         <t:perform-sort>
			         <t:sort select="doc/a" data-type="number"/>
		       </t:perform-sort>
      </out>
   </t:template>

   <t:template match="text()"/>
</t:transform>
