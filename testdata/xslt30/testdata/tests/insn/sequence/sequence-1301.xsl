<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: sequence constructor/concatenator with nodes -->

   <t:template match="/">
      <out>
         <a><t:value-of select="(//what,/doc/a)" separator=", "/></a>
         <b><t:value-of select="(//*/@attr,/doc/c/noodle,/doc/b)" separator=", "/></b>
      </out>
   </t:template>
</t:transform>
