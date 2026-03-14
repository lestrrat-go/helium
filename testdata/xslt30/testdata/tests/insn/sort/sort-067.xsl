<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test that case-order is ignored when collation is present (use codepoint collation for universality; with lower-first).  -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <With-collation>
            <t:for-each select="doc/item">
               <t:sort select="@target"
                  collation="http://www.w3.org/2005/xpath-functions/collation/codepoint" lang="en"
                  case-order="lower-first"/>
               <t:value-of select="@num"/>
               <t:text>:</t:text>
               <t:value-of select="@target"/>
               <t:text> * </t:text>
            </t:for-each>
         </With-collation>
         <Without-collation>
            <t:for-each select="doc/item">
               <t:sort select="@target" lang="en" case-order="lower-first"/>
               <t:value-of select="@num"/>
               <t:text>:</t:text>
               <t:value-of select="@target"/>
               <t:text> * </t:text>
            </t:for-each>
         </Without-collation>
      </out>
   </t:template>
</t:transform>
