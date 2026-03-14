<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:character-map with two different mappings of the same character, 
				one from @use-character-maps and one from xsl:output-character. -->

   <t:output use-character-maps="format2"/>

   <t:character-map name="format2" use-character-maps="format1">
      <t:output-character character="#" string="A"/>
      <t:output-character character="$" string="*"/>
   </t:character-map>

   <t:character-map name="format1">
      <t:output-character character="$" string="€"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
