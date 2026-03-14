<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" indent="no" encoding="UTF-8"/>
   <!-- Purpose: Test XHTML output method with @encoding=UTF-8. -->

   <t:template match="/">
      <html>
         <body>HelloÁ</body>
      </html>
   </t:template>
</t:transform>
