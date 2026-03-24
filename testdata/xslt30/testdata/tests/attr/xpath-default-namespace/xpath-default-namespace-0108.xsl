<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xpath-default-namespace declared on the element that itself contains XPath expressions with 
  				unprefixed element names inside a match attribute. -->

   <t:template match="/">
      <t:text>
</t:text>
      <out>
         <t:value-of select="doc/item[1]/subitem"/>
         <t:text>,</t:text>
         <t:value-of select="doc/item/attribute::attr"/>
         <t:text>,</t:text>
         <t:value-of select="doc/child::item/@attr"/>
         <t:for-each-group select="child::doc/*[name()='item']" group-starting-with="item">
    	       <t:text>
</t:text>
    	       <group>
               <t:value-of select="./*"/>
            </group>
         </t:for-each-group>
         <t:apply-templates select="*"/>
         <t:text>
</t:text>
      </out>
   </t:template>

   <t:template match="item|str" xpath-default-namespace="http://example.org/hal">
      <t:text>
</t:text>
      <elem>
         <t:number count="item | str"/>
         <t:text> </t:text>
         <t:value-of select="./*"/>
      </elem>
   </t:template>
</t:transform>
