<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: xsl:for-each over sequence of atomic values constructed by 'for' -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:for-each select="for $x in (10, 20, 30) return $x">
            <it>
               <t:value-of select="."/>
            </it>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
