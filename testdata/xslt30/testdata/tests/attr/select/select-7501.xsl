<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests xsl:for-each with missing the REQURED select attribute. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
	     <out>
         <t1>
            <t:for-each>
				           <t:value-of select="'error'"/>
			         </t:for-each>
         </t1>
      </out>
   </t:template>
</t:transform>
