<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests the default separator vs specified separator in an xsl:value-of instruction. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
   		<out>
         <t:text>
</t:text>
         <a>
            <t:value-of separator="_" select="doc/a"/>
         </a>
         <t:text>
</t:text>
         <b>
            <t:value-of select="doc/a"/>
         </b>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
