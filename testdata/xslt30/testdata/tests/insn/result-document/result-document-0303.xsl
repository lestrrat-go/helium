<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: xsl:result-document, JSON serialization; map-->

   <t:template name="t:initial-template">
      <t:result-document method="json">
         <t:map>
           <t:map-entry key="'a'" select="22"/>
           <t:map-entry key="'b'" select="23"/>
           <t:map-entry key="'c'" select="24"/>
         </t:map>
      </t:result-document>
   </t:template>
</t:transform>
