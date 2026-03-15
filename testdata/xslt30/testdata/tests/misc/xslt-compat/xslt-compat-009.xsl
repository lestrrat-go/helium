<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="1.0">
<!-- Purpose: Test of 1.0 stylesheet that has an unguarded instruction set to version 2.0, and using new 2.0 attribute of that instruction. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:attribute name="attrib" version="2.0" select="'Setting it the new way'"/>
      </out>
   </t:template>
</t:transform>
