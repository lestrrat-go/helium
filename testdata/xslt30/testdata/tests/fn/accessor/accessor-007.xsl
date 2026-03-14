<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of fn:document-uri accessor with node from a temporary tree as argument -->

   <t:template match="doc">
      <t:variable name="var1">
         <doc>top<t:processing-instruction name="pi">PI_data</t:processing-instruction>
            <t:comment>This is the 1st comment</t:comment>
            <t:text>text-in-doc</t:text>
            <t:element name="inner"/>
         </doc>
      </t:variable>
      <out>
         <t1>
            <t:value-of select="document-uri($var1/doc/processing-instruction()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <t:value-of select="document-uri($var1/doc/comment()) instance of empty-sequence()"/>
         </t2>
         <t3>
            <t:value-of select="document-uri($var1/doc/text()[1]) instance of empty-sequence()"/>
         </t3>
         <t4>
            <t:value-of select="document-uri($var1/doc/element()) instance of empty-sequence()"/>
         </t4>
         <t5>
            <t:value-of select="document-uri($var1/doc/attribute()) instance of empty-sequence()"/>
         </t5>
         <t6>
            <t:value-of select="document-uri($var1)"/>
         </t6>
      </out>
   </t:template>
</t:transform>
