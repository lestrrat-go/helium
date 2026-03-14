<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <t:output method="xml" indent="no"/>

   <!-- Purpose: An array is flattened with XML output method -->

   <t:template name="t:initial-template">
      <t:variable name="e1" as="element()"><e1 id="1"/></t:variable>
      <t:variable name="e2" as="element()"><e1 id="1"/></t:variable>
      <t:variable name="array" select="[$e1, $e2]"/>
      <out>
         <t:sequence select="$array"/>
      </out>
   </t:template>
</t:transform>
