<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   version="3.0">
   <!-- Purpose: Test character map support in HTML5-->
   <t:character-map name="map01">
      <t:output-character character="c" string="[C]"/>
      <t:output-character character="x" string="[X]"/>
   </t:character-map>
   <t:output method="html" version="5.0" html-version="5.0" escape-uri-attributes="yes"
      encoding="UTF-8" indent="yes"  use-character-maps="map01"/>
   <t:strip-space elements="*"/>

   <t:template match="/">
      <t:copy-of select="*"/>
   </t:template>

</t:transform>
