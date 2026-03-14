<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for one imported xsl:character-map, no character maps declared in primary stylesheet. -->

   <t:import href="x.xsl"/>

   <t:output use-character-maps="format1"/>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
