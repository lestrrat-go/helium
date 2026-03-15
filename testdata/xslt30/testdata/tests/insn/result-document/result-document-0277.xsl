<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test indent="yes", with XHTML result document where the stylesheet has a 
  				default namespace defined on the html element itself. -->

   <t:template match="doc">
      <t:result-document method="xhtml" encoding="UTF-8" indent="yes">
      <html xmlns="http://www.w3.org/1999/xhtml">
         <body>
            <t:text>This is the body</t:text>
         </body>
      </html>
      </t:result-document>
   </t:template>
</t:transform>
