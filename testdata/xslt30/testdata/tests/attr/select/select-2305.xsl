<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that the strings within the resulting sequence are 
  				concatenated, with a (zero-length) separator inserted between successive 
  				strings. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="str" select="(1 to 4)"/>

   <t:template match="/">
   		<out>
         <t:text>
</t:text>
         <a>
            <t:value-of>
					          <t:sequence select="('a','b','c','d')"/>
				        </t:value-of>
         </a>
         <t:text>
</t:text>
         <b>
            <t:value-of>
					          <t:sequence select="$str"/>
				        </t:value-of>
         </b>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
