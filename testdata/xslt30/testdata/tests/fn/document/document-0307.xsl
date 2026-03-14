<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that the result of doc(()) is an instance of document-node()? and empty-sequence().-->

   <t:template match="/doc">
      <out>
         <t:copy-of select="doc(()) instance of document-node()?"/>
         <t:copy-of select="doc(()) instance of empty-sequence()"/>
      </out>
   </t:template>
</t:transform>
