<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test XHTML result document with @byte-order-mark="no". Encoding is UTF-8.-->

   <t:template match="/">
      <t:result-document method="xhtml" 
                           indent="no" 
                           encoding="UTF-8" 
                           byte-order-mark="no">
         <html>
            <head>
               <title/>
            </head>
            <body>hello</body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
