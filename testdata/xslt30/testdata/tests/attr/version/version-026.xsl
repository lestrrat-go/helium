<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="8.5">
<!-- Purpose: Later-version (8.5) stylesheet has (fictional) 8.5 function guarded by use-when testing function-available, 2 arguments. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:value-of select="burnish('x',5)" use-when="function-available('burnish',2)"/>
         <t:value-of select="concat('fell back on ','burnish')"
                     use-when="not(function-available('burnish',2))"/>
      </out>
   </t:template>
</t:transform>
