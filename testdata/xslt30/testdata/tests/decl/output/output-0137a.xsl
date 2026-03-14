<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="3.0">

   <t:output method="xhtml"
             indent="no"
             encoding="UTF-8"
             byte-order-mark=" false "/>
   <!-- Purpose: Test XHTML output with @byte-order-mark=false. Encoding is UTF-8. -->

   <t:template match="/">
      <html>
         <body>Hello</body>
      </html>
   </t:template>
</t:transform>
