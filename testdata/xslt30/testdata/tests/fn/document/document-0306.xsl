<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that the result of fn:doc is a document-node().-->

   <t:template match="/doc">
      <out>
         <t:copy-of select="doc('doc02.xml') instance of document-node()"/>
      </out>
   </t:template>
</t:transform>
