<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that pattern "document-node(element(E))" gets a default priority of 0. -->

   <t:output method="xml" indent="no" encoding="UTF-8"/>

   <t:template match="document-node(element(doc))">
      <out>
         <t:value-of select="doc/a/@x5"/>
      </out>
   </t:template>

   <t:template match="document-node(element(doc))" priority="0.1">
      <out>
         <t:value-of select="doc/a/@x4"/>
      </out>
   </t:template>

   <t:template match="document-node(element(doc))" priority="-0.1">
      <out>
         <t:value-of select="doc/a/@x6"/>
      </out>
   </t:template>
</t:transform>
