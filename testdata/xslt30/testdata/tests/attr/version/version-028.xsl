<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="8.5">
   <!-- Purpose: Later-version (8.5) stylesheet has (fictional) 8.5 attribute on older instruction, unguarded. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:for-each-group select="doc/num" group-adjacent="ceiling(position() div 4)"
            new-8.5-trick="4">
            <a key="{current-grouping-key()}">
               <b>
                  <t:value-of select="current-group()/@p" separator=","/>
               </b>
            </a>
         </t:for-each-group>
      </out>
   </t:template>
</t:transform>
