<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="xml" indent="no" encoding="UTF-8" byte-order-mark="yes"/>
   <!-- Purpose: Test XML output with @byte-order-mark=yes. Encoding is UTF-8. -->

   <t:template match="/">
      <html>
         <body>Hello</body>
      </html>
   </t:template>
</t:transform>
