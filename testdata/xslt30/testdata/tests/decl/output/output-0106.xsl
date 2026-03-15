<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with XHTML output where the stylesheet has a 
  				default namespace defined on the html element itself. -->

   <t:output method="xhtml" encoding="UTF-8" indent="yes"/>

   <t:template match="doc">
      <html xmlns="http://www.w3.org/1999/xhtml">
         <body>
            <t:text>This is the body</t:text>
         </body>
      </html>
   </t:template>
</t:transform>
