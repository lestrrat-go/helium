<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of nilled accessor with empty sequence as argument -->

   <t:template match="doc">
	     <out>
         <t1>
            <t:value-of select="nilled(()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <t:value-of select="nilled(./*) instance of empty-sequence()"/>
         </t2>
      </out>
	  </t:template>
</t:transform>
