<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to verify that implicit default collation is captured by fn:default-collation -->

   <t:template match="doc">
      <out>
         <t:value-of select="default-collation()"/>
      </out> 
   </t:template>
</t:transform>
