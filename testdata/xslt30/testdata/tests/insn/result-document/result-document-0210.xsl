<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document with missing @method. Allow XHTML output method to be determined by default rules.-->

   <t:template match="/">
      <t:result-document>
         <html xmlns="http://www.w3.org/1999/xhtml">
            <head>
               <title/>
            </head>
            <body>hello</body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
