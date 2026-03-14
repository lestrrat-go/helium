<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example1.com"
             xmlns:one="http://example2.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="my one">
<!-- Purpose: Test @format of xsl:result-document. Verify that the matching 
  				output declaration is used. (output declarations have same local 
  				name but different namespace, namespace in-scope of xsl:result-document 
  				determines which one will be used) -->

   <t:output name="one:format2" method="text" encoding="UTF-8"/>

   <t:output name="my:format2" method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <t:result-document xmlns:one="http://example1.com" format="one:format2">
         <out>This document should be serialized as xml, not as text because the named my:format has method "xml".</out>
      </t:result-document>
   </t:template>
</t:transform>
