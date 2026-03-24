<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:hal="http://example.org/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="hal"
             version="2.0">
<!--            Test with xpath-default-namespace declared on an ancestor(not parent or xsl:stylesheet) 
  				of the element that contains XPath expressions with unprefixed element names in:
				- @select of xsl:value-of, xsl:apply-templates, xsl:with-param
				- @count of xsl:number
				- @group-starting-with of xsl:for-each-group -->

   <t:template match="/">
      <out t:xpath-default-namespace="http://example.org/hal"><result>
            <t:value-of select="doc/item[1]/subitem"/>
            <t:text>,</t:text>
            <t:value-of select="doc/item/attribute::attr"/>
            <t:text>,</t:text>
            <t:value-of select="doc/child::item/@attr"/>
            <t:for-each-group select="child::doc/*[name()='item']" group-starting-with="item">
               <group>
                  <t:call-template name="new">
    			              <t:with-param name="p" select="//item[2]/subitem"/>
    			           </t:call-template>
               </group>
    	       </t:for-each-group>
            <t:apply-templates select="doc/item|doc/str"/>
         </result>
      </out>
   </t:template>

   <t:template name="new">
	     <t:param name="p"/>
	     <t:value-of select="$p"/>
   </t:template>

   <t:template match="hal:item|hal:str">
</t:template>
</t:transform>
