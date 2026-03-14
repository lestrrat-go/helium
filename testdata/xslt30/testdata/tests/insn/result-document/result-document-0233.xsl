<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document and @use-character-maps.-->

   <t:character-map name="temp">
      <t:output-character character="a" string="b"/>
      <t:output-character character="&#xA;" string=" "/>
      <t:output-character character="«" string="S"/> 
   </t:character-map>

   <t:template match="/">
  	   <t:result-document method="xml" use-character-maps="temp">
      	  <out>baba«ba«</out>
      </t:result-document>
   </t:template>
</t:transform>
