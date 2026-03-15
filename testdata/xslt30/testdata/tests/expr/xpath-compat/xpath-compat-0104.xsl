<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test integer being divided by zero; returns infinity in 1.0 behavior. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:value-of select="format-number(1 div 0,'#######################')"/>
      </out>
   </t:template>
</t:transform>
