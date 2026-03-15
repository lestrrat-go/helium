<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: xsl:result-document, adaptive serialization; -->

   <t:template name="t:initial-template">
      <t:result-document method="adaptive" item-separator="|" build-tree="false" omit-xml-declaration="true">
         <t:map>
           <t:map-entry key="'a'" select="22"/>
         </t:map>
         <elem/>
         <t:attribute name="a" select="5"/>
      </t:result-document>
   </t:template>
</t:transform>
