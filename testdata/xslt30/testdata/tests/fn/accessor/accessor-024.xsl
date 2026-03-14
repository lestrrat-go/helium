<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test base-uri after a node has been copied -->

   <t:template match="doc">
	
	     <t:variable name="v1">
		       <t:copy-of select="/doc"/>
	     </t:variable>
  
	     <out>
         <t1>
            <t:value-of select="ends-with(base-uri($v1/doc/processing-instruction()),'accessor-024.xsl') "/>
         </t1>
         <t2>
            <t:value-of select="base-uri($v1/doc/comment()[1]) instance of empty-sequence()"/>
         </t2>
         <t3>
            <t:value-of select="base-uri($v1/doc/text()[1]) instance of empty-sequence()"/>
         </t3>
         <t4>
            <t:value-of select="ends-with(base-uri($v1/doc/element()), 'accessor-024.xsl')"/>
         </t4>
         <t5>
            <t:value-of select="base-uri($v1/doc/attribute()) instance of empty-sequence()"/>
         </t5>
         <t6>
            <t:value-of select="ends-with(base-uri($v1), 'accessor-024.xsl')"/>
         </t6>
      </out>
   </t:template>
</t:transform>
