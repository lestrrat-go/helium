<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <t:output method="json" indent="no" json-node-output-method="html"/>

   <!-- Purpose: Test JSON serialization, maps with nested HTML -->

   <t:template name="t:initial-template">
      <t:map>
         <t:map-entry key="'doc1'">
            <html><head><title>Document 1</title></head><body><p>Content 1</p></body></html>
         </t:map-entry>
         <t:map-entry key="'doc2'">
            <html><head><title>Document 2</title></head><body><p>Content 2</p></body></html>
         </t:map-entry>
      </t:map>
   </t:template>
</t:transform>
