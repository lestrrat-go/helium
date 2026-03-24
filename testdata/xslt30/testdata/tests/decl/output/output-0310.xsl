<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Show that text node children of elements present in @cdata-section-elements 
  				are not affected by character maps.-->

   <t:output method="xml"
             indent="no"
             encoding="UTF-8"
             cdata-section-elements="b"
             use-character-maps="format1 format2"/>

   <t:character-map name="format1">
      <t:output-character character="a" string="A"/>
   </t:character-map>

   <t:character-map name="format2">
      <t:output-character character="#" string="*"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <a>aaa</a>
         <b>aaa</b>
         <t:text/>
         <t:text>
</t:text>
         <t:for-each select="a">
    	       <a>
               <t:value-of select="."/>
            </a>
         </t:for-each>
         <t:text>
</t:text>
         <t:for-each select="a">
    	       <b>
               <t:value-of select="."/>
            </b>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
