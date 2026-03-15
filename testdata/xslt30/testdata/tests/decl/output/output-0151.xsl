<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test empty elements with non-empty content model in XHTML output. -->

   <t:output method="xhtml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
      <t:text>
</t:text>
      <html>
         <t:text>
</t:text>
         <head>
            <title/>
         </head>
         <t:text>
</t:text>
         <body>
            <p/>
         </body>
         <t:text>
</t:text>
      </html>
   </t:template>
</t:transform>
