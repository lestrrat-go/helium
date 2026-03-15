<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result document with no attributes. Output is controlled by unnamed xsl:output declaration. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <t:result-document>
         <out>In XML doc with US-ASCII.</out>
      </t:result-document>
   </t:template>
</t:transform>
