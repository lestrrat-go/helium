<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test for using a list of character maps with the XML output method. -->

   <t:output method="xml"
             indent="no"
             encoding="UTF-8"
             use-character-maps="format1 format2 format3"/>

   <t:character-map name="format1">
      <t:output-character character="$" string="A"/>
   </t:character-map>

   <t:character-map name="format2">
      <t:output-character character="#" string="&amp;"/>
   </t:character-map>

   <t:character-map name="format3">
      <t:output-character character="*" string="+"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:text>
</t:text>
         <t:for-each select="*">
    	       <a>
               <t:value-of select="."/>
            </a>
            <t:text>
</t:text>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
