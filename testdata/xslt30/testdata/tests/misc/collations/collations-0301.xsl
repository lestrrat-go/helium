<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: To test the xsl:sort element for primary and secondary sorting with attributes that affect collation  -->

   <t:template match="w3cgroup">
      <out>
         <t:apply-templates select="member">
            <t:sort select="primary/name/first" lang="en" case-order="lower-first"/>
            <t:sort select="primary/name/last"
                    order="ascending"
                    lang="en"
                    collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
         </t:apply-templates>
      </out>
   </t:template>

   <t:template match="member">
      <t:value-of select="primary/name/first"/>
      <t:text> </t:text>
      <t:value-of select="primary/name/last"/>
      <t:text>; </t:text>
   </t:template>
</t:transform>
