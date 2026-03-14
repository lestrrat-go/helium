<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with two character maps with the same name but 
  				with different import precedence. Show that the one 
  				with higher precedence is used.-->

   <t:import href="x.xsl"/>

   <t:output use-character-maps="format1"
             method="xml"
             indent="no"
             encoding="UTF-8"/>

   <t:character-map name="format1">
      <t:output-character character="$" string="*"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
