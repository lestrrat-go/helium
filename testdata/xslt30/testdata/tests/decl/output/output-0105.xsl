<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with XHTML output method. The document element name is 'html' in the null namespace.  -->

   <t:output method="xhtml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
      <html>
         <body>
            <t:text>This is the body</t:text>
         </body>
      </html>
   </t:template>
</t:transform>
