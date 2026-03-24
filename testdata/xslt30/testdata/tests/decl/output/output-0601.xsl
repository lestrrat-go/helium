<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   version="3.0">
   <!-- Purpose: Test namespace prefixing of SVG/MathML etc in HTML5-->

   <t:output method="html" version="5.0" html-version="5.0" escape-uri-attributes="yes"
      encoding="UTF-8" indent="yes"/>
   <t:strip-space elements="*"/>

   <t:template match="/">
      <t:copy-of select="*"/>
   </t:template>

</t:transform>
