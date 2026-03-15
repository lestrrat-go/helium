<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Testcase to verify that base-uri() and base-uri(.) return the same result.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="//substring1"/>
      </out>
   </t:template>

   <t:template match="substring1">
      <t:value-of select="base-uri() eq base-uri(.)"/>
   </t:template>
</t:transform>
