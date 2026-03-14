<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test for xsl:character-map and @use-character-maps defined on it. -->

   <t:output use-character-maps="format2"
             method="xml"
             indent="no"
             encoding="UTF-8"/>

   <t:character-map name="format2" use-character-maps="format1"/>

   <t:character-map name="format1">
      <t:output-character character="$" string="€"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
