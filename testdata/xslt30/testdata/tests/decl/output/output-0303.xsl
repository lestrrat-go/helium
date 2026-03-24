<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test for using character maps with the TEXT output method. -->

   <t:output method="text"
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
      <html>
         <body>
            <t:for-each select="*">
    	          <p>
                  <t:value-of select="."/>
               </p>
               <t:text>
</t:text>
            </t:for-each>
         </body>
      </html>
   </t:template>
</t:transform>
