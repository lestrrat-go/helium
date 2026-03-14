<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test error code attribute.  -->

   <t:template match="docs">
      <out>
         <t:message select="'Error message'" terminate="yes" error-code="UIOP9876"/>
      </out>
   </t:template>
</t:transform>
