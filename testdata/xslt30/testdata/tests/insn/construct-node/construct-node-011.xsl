<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses the separator attribute on an xsl:value-of to separate 
  				adjacent items in the result sequence. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:value-of separator="_"> 
  				      <t:sequence select="1 to 5"/>
			      </t:value-of>
         <t:text> </t:text>
         <t:value-of separator="*_*" select="2 to 6"/>
      </out>
	  </t:template>
</t:transform>
