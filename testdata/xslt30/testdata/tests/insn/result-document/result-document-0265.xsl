<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test escaping of non-ASCII characters in HTML result document when @escape-uri-attributes="true".-->

   <t:template match="/">
      <t:result-document method="html"
         escape-uri-attributes="true"
         encoding="UTF-8"
         indent="no">
      <html>
         <body>
            <t:text>
</t:text>
            <div>
This is <a href="http://iri.example.org/ﭏ/årsrapport/år/2005?x=y"> escaped</a>
            </div>
         </body>
      </html>
      </t:result-document>
   </t:template>
</t:transform>
