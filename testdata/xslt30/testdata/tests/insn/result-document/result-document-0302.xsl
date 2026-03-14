<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: build-tree=no on xsl:result-document.-->

   <t:template match="/">
      <t:result-document build-tree="no" item-separator="|" method="text">
         <t:sequence select="1 to count(//*)"/>
      </t:result-document>
   </t:template>
</t:transform>
