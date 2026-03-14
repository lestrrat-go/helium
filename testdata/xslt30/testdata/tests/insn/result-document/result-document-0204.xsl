<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:  Test that value of @href in xsl:result-document can be set to zero-length string. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <t:result-document href="">
         <out>Empty href on result-document.</out>
      </t:result-document>
   </t:template>
</t:transform>
