<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test two AVTs with literal element between them in the 
  				separator attribute of xsl:value-of.-->

   <t:template match="/doc">
      <out>
         <t:value-of select="str" separator="{item[1]};{item[2]}"/>
      </out>
   </t:template>
</t:transform>
