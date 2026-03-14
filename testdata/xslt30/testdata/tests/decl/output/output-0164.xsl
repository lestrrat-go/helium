<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test HTML output with default value for @escape-uri-attributes. (default is 'yes') -->

   <t:output method="html" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <html>
         <body>
            <div>This is <a href="http://iri.example.org/ﭏ/årsrapport/år/2005?x=y"> escaped</a>
            </div>
         </body>
      </html>
   </t:template>
</t:transform>
