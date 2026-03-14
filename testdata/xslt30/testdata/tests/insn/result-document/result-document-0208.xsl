<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document with missing @method. Allow XML output method to be determined by default rules.-->

   <t:template match="/">
      <t:result-document>
         <html xmlns="http://www.mytest.example.org">
            <out>hello</out>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
