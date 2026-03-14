<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for two xsl:result-document instructions both using the principal output URI -->

   <t:output method="xhtml"/>

   <t:template match="/">
      <t:result-document>
         <html>
            <head/>
            <body>First HTML doc in implict xsl:result-document</body>
         </html>
      </t:result-document>
      <t:result-document href="">
         <html>
            <head/>
            <body>Second HTML doc in explicit xsl:result-document</body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
