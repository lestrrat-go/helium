<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that the sequence in an xsl:value-of instruction is atomized. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="v1" as="element() *">
		    <t:sequence select="doc/a"/>
	  </t:variable>

   <t:variable name="v2">
		    <t:sequence select="doc/a"/>
	  </t:variable>

   <t:template match="/">
   		<out>
         <t:text>
</t:text>
         <a>
            <t:value-of select="doc/a"/>
         </a>
         <t:text>
</t:text>
         <b>
            <t:value-of select="$v1"/>
         </b>
         <t:text>
</t:text>
         <b>
            <t:value-of select="$v2"/>
         </b>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
