<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of node-name accessor with copied nodes as argument -->

   <t:template match="/">
	
	     <t:variable name="var1">
	        <t:copy-of select="/"/>
  	   </t:variable>
  
	     <out>
         <t1>
            <t:value-of select="node-name($var1/doc/processing-instruction())"/>
         </t1>
         <t2>
            <t:value-of select="node-name($var1/doc/comment())"/>
         </t2>
         <t3>
            <t:value-of select="node-name($var1/doc/text()[1])"/>
         </t3>
         <t4>
            <t:value-of select="node-name($var1/doc/element())"/>
         </t4>
         <t5>
            <t:value-of select="node-name($var1/doc/attribute())"/>
         </t5>
         <t6>
            <t:value-of select="node-name($var1)"/>
         </t6>
         <t7>
            <t:for-each select="$var1/doc/namespace::*[name(.) = 'mynamespace']">
				           <t:value-of select="node-name(.)"/>
			         </t:for-each>
         </t7>
      </out>
   </t:template>
</t:transform>
