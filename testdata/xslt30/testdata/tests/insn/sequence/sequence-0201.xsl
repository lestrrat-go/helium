<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: xsl:for-each over sequence of atomic values -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:for-each select="(10, 20, 30)">
            <it>
               <t:value-of select="."/>
            </it>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
