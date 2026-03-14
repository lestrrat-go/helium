<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://example.org/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Testcase to verify that the default namespace of the stylesheet is not used for unprefixed elements 
  				in Xpath expressions and the only way to set a default namespace for them is by use of xpath-default-namespace. -->

   <t:template match="/">
      <out >
         <t:value-of select="doc/item[1]/subitem"/>
         <t:text>,</t:text>
         <t:value-of select="doc/item/attribute::attr"/>
         <t:text>,</t:text>
         <t:value-of select="doc/child::item/@attr"
                     xpath-default-namespace="http://example.org/hal"/>
      </out>
   </t:template>
</t:transform>
