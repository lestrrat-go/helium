<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://example.org/hal"
             xmlns:my="http://example.org/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with xpath-default-namespace declared and then set to a different value for some elements. -->

   <t:template match="/" xpath-default-namespace="http://example.org/hal">
      <out>
         <E1>
            <t:value-of select="doc/str/*"/>
            <t:text>,</t:text>
            <t:value-of select="//substr[2]"/>
            <t:text>,</t:text>
            <t:value-of select="doc/item[1]/subitem"/>
            <t:text>,</t:text>
            <t:value-of select="doc/item/attribute::attr"/>
            <t:text>,</t:text>
            <t:value-of select="doc/child::item/@attr"/>
         </E1>
         <E2 t:xpath-default-namespace="http://ns.example.org/test"><t:value-of select="my:doc/str/*"/>
            <t:text>,</t:text>
            <t:value-of select="//substr[2]"/>
            <t:text>,</t:text>
            <t:value-of select="my:doc/item[1]/subitem"/>
            <t:text>,</t:text>
            <t:value-of select="my:doc/item/attribute::attr"/>
            <t:text>,</t:text>
            <t:value-of select="my:doc/child::item/@attr"/>
         </E2>
      </out>
   </t:template>
</t:transform>
