<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test of xsl:next-match and no other matching templates are found. 
  				Verifies that built-in template rules are invoked. 
            Uses shallow-copy built-in rule-set -->
   
   <t:mode on-no-match="deep-skip"/>

   <t:template match="doc">
      <out>
         <t:apply-templates select="*">
            <t:with-param name="p1" select="'top'"/>
         </t:apply-templates>
      </out>
   </t:template>

   <t:template match="tag">
      <tag>
         <t:next-match/>
      </tag>
   </t:template>

   <t:template match="bag">
      <bag>
         <t:next-match/>
      </bag>
   </t:template>
</t:transform>
