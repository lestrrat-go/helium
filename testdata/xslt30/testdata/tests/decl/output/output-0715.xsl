<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <!-- Purpose: An array is flattened with dynamically-selected output method -->

   <t:template name="t:initial-template">
      <t:variable name="e1" as="element()"><head><title>A document</title></head></t:variable>
      <t:variable name="e2" as="element()"><body><p>Some content</p></body></t:variable>
      <t:variable name="array" select="[$e1, $e2]"/>
      <html>
         <t:sequence select="$array"/>
      </html>
   </t:template>
</t:transform>
