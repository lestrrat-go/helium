<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document and relative @href value.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <t:result-document href="result-document-0211b.out">
         <out>hello</out>
      </t:result-document>
   </t:template>
</t:transform>
