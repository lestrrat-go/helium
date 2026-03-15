<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test id() first argument choosing all nodes even in version=1.0 scope. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:for-each select="id(main/b)">
            <t:value-of select="."/>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
