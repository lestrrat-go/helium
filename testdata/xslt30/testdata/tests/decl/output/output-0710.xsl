<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <t:output method="xml" indent="no"/>

   <!-- Purpose: Serializing a map fails with XML output method -->

   <t:template name="t:initial-template">
      <t:variable name="maps" as="map(*)*">
         <t:for-each select="1 to 5">
            <t:map>
               <t:map-entry key="'index'">
                  <t:sequence select="."/>
               </t:map-entry>
               <t:map-entry key="'value'">
                  <t:sequence select=".*."/>
               </t:map-entry>
            </t:map>
         </t:for-each>
      </t:variable>
      <t:sequence select="$maps"/>     
   </t:template>
</t:transform>
