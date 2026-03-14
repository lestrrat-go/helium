<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <t:output method="json" indent="no" allow-duplicate-names="true" build-tree="false"/>

   <!-- Purpose: Test JSON serialization, maps with allow-duplicate-names=yes -->

   <t:template name="t:initial-template">
      <t:map>
         <t:map-entry key="xs:time('23:00:00Z')">
            <t:sequence select="'alpha'"/>
         </t:map-entry>
         <t:map-entry key="'23:00:00Z'">
            <t:sequence select="'beta'"/>
         </t:map-entry>
      </t:map>
   </t:template>
</t:transform>
