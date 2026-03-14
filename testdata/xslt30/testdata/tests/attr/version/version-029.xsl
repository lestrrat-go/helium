<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: 2.0 stylesheet has (fictional) higher-versioned instruction guarded by xsl:fallback. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:new-8.5-feature version="8.5">
            <t:fallback>
               <t:text>we fell back</t:text>
            </t:fallback>
            <t:text>should NOT appear</t:text>
         </t:new-8.5-feature>
         <t:for-each-group select="doc/num" group-adjacent="ceiling(position() div 4)">
            <a key="{current-grouping-key()}">
               <b>
                  <t:value-of select="current-group()/@p" separator=","/>
               </b>
            </a>
         </t:for-each-group>
      </out>
   </t:template>
</t:transform>
