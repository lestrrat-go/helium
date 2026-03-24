<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test AVT in @indent of xsl:result-document, which evaluates to " true ". -->

   <t:template match="/">
      <t:result-document method="xhtml" encoding="UTF-8" indent="{doc/foo}">
      <html xmlns="http://www.w3.org/1999/xhtml">
         <body>
            <t:text>This is the body</t:text>
         </body>
      </html>
      </t:result-document>
   </t:template>
</t:transform>
