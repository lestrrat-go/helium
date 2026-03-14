<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of nilled accessor with copied nodes as argument -->

   <t:template match="/">
	
	     <t:variable name="var1">
	        <t:copy-of select="/"/>
      </t:variable>
  
	     <out>
         <t1>
            <t:value-of select="nilled($var1/doc/processing-instruction()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <t:value-of select="nilled($var1/doc/comment()) instance of empty-sequence()"/>
         </t2>
         <t3>
            <t:value-of select="nilled($var1/doc/text()[1]) instance of empty-sequence()"/>
         </t3>
         <t4>
            <t:value-of select="nilled($var1/doc/element())"/>
         </t4>
         <t5>
            <t:value-of select="nilled($var1/doc/attribute()) instance of empty-sequence()"/>
         </t5>
         <t6>
            <t:value-of select="nilled($var1) instance of empty-sequence()"/>
         </t6>
      </out>
   </t:template>
</t:transform>
