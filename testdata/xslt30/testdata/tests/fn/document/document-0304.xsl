<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to verify that two calls of doc($uri) return the 
  				same document node if the same $uri is supplied to both calls.-->

   <t:template match="/">
      <out>
         <t:value-of select="doc('doc03.xml') is doc('doc03.xml')"/>
      </out>
   </t:template>
</t:transform>
