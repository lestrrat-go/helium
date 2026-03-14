<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example1.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test that value in @use-character-maps of xsl:result-document both overrides 
				and supplements such values in a matching output definition.-->

   <t:character-map name="temp">
      <t:output-character character="a" string="A"/>
      <t:output-character character="&#xA;" string="XX"/>
   </t:character-map>

   <t:character-map name="two">
      <t:output-character character="b" string="B"/>
      <t:output-character character="&#xA;" string=" "/>
      <t:output-character character="«" string="S"/> 
   </t:character-map>

   <t:output name="my:temp-output"
             method="xml"
             encoding="UTF-8"
             indent="no"
             version="1.0"
             use-character-maps="temp"/>

   <t:template match="/">
  	   <t:result-document format="my:temp-output" method="xml" use-character-maps="two">
      	  <out>baba«ba«</out>
      </t:result-document>
   </t:template>
</t:transform>
