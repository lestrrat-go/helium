<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Range expression, variable to variable (both integers) -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="lo" select="-3"/>

   <t:variable name="hi" select="2"/>

   <t:template match="/">
      <out>
         <t:value-of select="($lo to $hi)" separator=", "/>
      </out>
   </t:template>
</t:transform>
