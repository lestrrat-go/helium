<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Use named output format on secondary output -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:output name="my:format2" method="text" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <log>Before redirect</log>
         <t:result-document href="multresult2.out" format="my:format2">
            <t:text>This should come out as plain text.</t:text>
         </t:result-document>
         <log>After redirect</log>
      </out>
   </t:template>
</t:transform>
