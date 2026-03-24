<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses an ifexpr in @select of an xsl:sort instruction.(Using count function in the ifexpr) -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:for-each select="doc/alpha">
            <t:sort select="if (true()) then count(following-sibling::*) else count(preceding-sibling::*) "
                    data-type="number"/>
            <t:value-of select="@key"/>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
