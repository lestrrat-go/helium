<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xpath-default-namespace="http://www.example.com/hal"
             version="2.0">

   <!-- Show that @xpath-default-namespace does not affect attribute names.-->

   <t:template match="/">
      <out>
         <t:value-of select="doc/item/@attr"/>
      </out>
   </t:template>
</t:transform>
