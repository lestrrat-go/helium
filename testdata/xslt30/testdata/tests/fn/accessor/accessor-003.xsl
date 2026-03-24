<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of node-name() accessor with node stored in variable -->

   <t:template match="doc">
      <out>
         <t1>
            <t:variable name="var1" select="processing-instruction()"/>
            <t:value-of select="node-name($var1)"/>
         </t1>
         <t2>
            <t:variable name="var2" select="comment()"/>
            <t:value-of select="node-name($var2)"/>
         </t2>
         <t3>
            <t:variable name="var3" select="text()[1]"/>
            <t:value-of select="node-name($var3)"/>
         </t3>
         <t4>
            <t:variable name="var4" select="element()"/>
            <t:value-of select="node-name($var4)"/>
         </t4>
         <t5>
            <t:variable name="var5" select="attribute()"/>
            <t:value-of select="node-name($var5)"/>
         </t5>
         <t6>
            <t:variable name="var6" select="namespace::*"/>
            <t:for-each select="$var6[name(.)='mynamespace']">
			            <t:value-of select="node-name(.)"/>
		          </t:for-each>
         </t6>
      </out>
   </t:template>
</t:transform>
