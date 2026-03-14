<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that pattern "element()" gets a default priority of -0.5. -->

   <t:output method="xml" indent="no" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="element()">
      <t:text>Match-of-element-no-name:</t:text>
      <t:value-of select="name(.)"/>
   </t:template>

   <t:template match="boo" priority="-0.4">
      <t:text>Match-boo</t:text>
   </t:template>

   <t:template match="foo" priority="-0.6">
      <t:text>Match-foo</t:text>
   </t:template>
</t:transform>
