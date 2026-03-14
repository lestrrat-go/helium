<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that result of collection($arg) is of type node()*. -->

   <t:template match="/">
      <out>
         <t:value-of select="collection('catalog1.xml') instance of node()*"/>
      </out>
   </t:template>
</t:transform>
