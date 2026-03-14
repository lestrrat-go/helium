<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Switch to secondary output and back to primary - sequentially. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <log>Before redirect</log>
         <t:result-document href="multresult1.out">
            <t:copy-of select="foo"/>
         </t:result-document>
         <log>After redirect</log>
      </out>
   </t:template>
</t:transform>
