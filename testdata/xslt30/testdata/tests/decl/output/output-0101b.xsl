<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test escaping of non-ASCII characters in HTML output when @escape-uri-attributes="1".-->

   <t:output method="html"
             escape-uri-attributes="1"
             encoding="UTF-8"
             indent="no"/>

   <t:template match="/">
      <html>
         <body>
            <t:text>
</t:text>
            <div>
This is <a href="http://iri.example.org/ﭏ/årsrapport/år/2005?x=y"> escaped</a>
            </div>
         </body>
      </html>
   </t:template>
</t:transform>
