<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example.org/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!--            Test with xpath-default-namespace declared on the parent of an element that contains 
  				XPath expressions with unprefixed element names in:
				- @select of xsl:value-of and xsl:apply-templates
				- @group-starting-with of xsl:for-each-group -->

   <t:template match="/">
      <out t:xpath-default-namespace="http://example.org/hal"><t:value-of select="doc/item[1]/subitem"/>
         <t:text>,</t:text>
         <t:value-of select="doc/item/attribute::attr"/>
         <t:text>,</t:text>
         <t:value-of select="doc/child::item/@attr"/>
         <t:for-each-group select="child::doc/*[name()='item']" group-starting-with="item">
    	    <group>
               <t:value-of select="./*"/>
            </group>
         </t:for-each-group>
         <t:apply-templates select="doc/item|doc/str"/>
      </out>
   </t:template>

   <t:template match="item|str">
      <elem>
         <t:number count="item | str"/>
         <t:text> </t:text>
         <t:value-of select="//item[2]/subitem"/>
      </elem>
   </t:template>

   <t:template match="my:item|my:str">
</t:template>
</t:transform>
