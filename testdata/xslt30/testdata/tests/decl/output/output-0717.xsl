<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <!-- Purpose: JSON node-output-method = xml-->
   
   <t:output method="json" json-node-output-method="xml"/>

   <t:template name="t:initial-template">
      <t:variable name="e1" as="element()"><head><title>A document</title></head></t:variable>
      <t:variable name="e2" as="element()"><body><p>Some content</p></body></t:variable>
      <t:variable name="array" select="[$e1, $e2]"/>
      <t:sequence select="$array"/>
   </t:template>
</t:transform>
