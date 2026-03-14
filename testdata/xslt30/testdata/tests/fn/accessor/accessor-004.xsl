<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of node-name() accessor with node from a temporary tree -->

   <t:template match="doc">
      <t:variable name="var1">
         <t:element name="doc">
            <t:attribute name="att">
               <t:text>top</t:text>
            </t:attribute>
            <t:processing-instruction name="pi">PI_data</t:processing-instruction>
            <t:comment>This is the 1st comment</t:comment>
            <t:text>text-in-doc</t:text>
            <t:element name="inner"/>
         </t:element>
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
            <t:for-each select="$var1/doc/namespace::*">
               <t:value-of select="node-name(.)"/>
            </t:for-each>
         </t6>
      </out>
   </t:template>
</t:transform>
