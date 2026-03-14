<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Testcase with fn:base-uri($arg) on six kinds of nodes: processing-instruction, 
  				comment node, element node, text node, attribute node, namespace node. Base-uri 
  				property is declared on nodes themselves or on their ancestors. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:strip-space elements="*"/>

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <t:text>Element node substring1: </t:text>
         <t:value-of select="base-uri(//substring1)"/>
         <t:text>
</t:text>
         <t:text>Attribute node attr2: </t:text>
         <t:value-of select="base-uri(//@attr2)"/>
         <t:text>
</t:text>
         <t:text>Comment node: </t:text>
         <t:value-of select="base-uri(doc/comment())"/>
         <t:text>
</t:text>
         <t:text>PI node: </t:text>
         <t:value-of select="base-uri(//processing-instruction())"/>
         <t:text>
</t:text>
         <t:text>Text node: </t:text>
         <t:value-of select="base-uri(doc/str2/text())"/>
         <t:text>
</t:text>
         <t:text>Namespace node: </t:text>
         <t:value-of select="base-uri(doc/namespace::*[2])"/>
         <t:text>
</t:text>
      </out>
   </t:template>
</t:transform>
