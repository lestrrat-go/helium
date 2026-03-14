<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase where doc-available($uri) returns true. Both relative and absolute URI values are tested for $uri.-->

   <t:variable name="var" select="/doc/str1"/>

   <t:template match="/doc">
      <out>
         <t:value-of select="doc-available($var)"/>
         <t:value-of select="doc-available('doc04.xml')"/>
         <t:value-of select="doc-available(resolve-uri($var, static-base-uri()))"/>
      </out>
   </t:template>
</t:transform>
