<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xpath-default-namespace="http://example.org/hal" version="2.0">
   <!-- Purpose: Test with xpath-default-namespace attribute declared on the xsl:stylesheet element. -->

   <t:template match="/">
      <out>
         <alpha>
            <t:value-of select="doc/item[1]/subitem"/>
            <t:text>,</t:text>
            <t:value-of select="doc/item/attribute::attr"/>
            <t:text>,</t:text>
            <t:value-of select="doc/child::item/@attr"/>
         </alpha>
         <t:for-each-group select="child::doc/*[name()='item']" group-starting-with="item">
            <group>
               <t:value-of select="./*"/>
            </group>
         </t:for-each-group>
         <beta>
            <t:apply-templates select="doc/item|doc/str"/>
         </beta>
      </out>
   </t:template>

   <t:template match="item|str">
      <elem>
         <t:number count="item | str"/>
         <t:text> </t:text>
         <t:value-of select="./*"/>
      </elem>
   </t:template>
</t:transform>
