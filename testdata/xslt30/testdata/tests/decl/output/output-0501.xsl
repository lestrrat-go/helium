<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for missing name in xsl:character-map. Static error -->

   <t:character-map>
      <t:output-character character="&#xA;" string="&#x9;"/>
   </t:character-map>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
