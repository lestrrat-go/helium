<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of xsl:next-match in a template where the pattern has a 
  				union operator with *:NCName Name Test and element(QName) that 
  				match one and the same node. Template has assigned priority of 1. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
      <out>
         <t:apply-templates select="x"/>
      </out>
   </t:template>

   <t:template match="*:x|element(x)" priority="1">
      <foo xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
      <t:next-match/>
   </t:template>

   <t:template match="node()">
      <bar xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
   </t:template>
</t:transform>
