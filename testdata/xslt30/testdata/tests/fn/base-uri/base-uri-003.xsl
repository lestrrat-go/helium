<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Testcase with fn:resolve-uri($relative) where $relative is an empty string.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:text>
</t:text>
         <out>
            <t:value-of select="resolve-uri('')"/>
         </out>
         <t:text>
</t:text>
         <out>
            <t:value-of select="resolve-uri(string(.))"/>
         </out>
      </output>
   </t:template>
</t:transform>
