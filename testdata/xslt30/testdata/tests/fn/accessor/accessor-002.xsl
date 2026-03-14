<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of node-name() accessor with empty sequence as argument -->

   <t:template match="/">
	     <out>
         <t1>
            <t:value-of select="node-name(()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <t:value-of select="node-name(doc/inner/*) instance of empty-sequence()"/>
         </t2>
      </out>
	  </t:template>
</t:transform>
