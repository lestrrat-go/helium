<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test for using one character map with the XML output method. -->

   <t:output method="xml"
             indent="no"
             encoding="UTF-8"
             use-character-maps="format1"/>

   <t:character-map name="format1">
      <t:output-character character="$" string="€"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
