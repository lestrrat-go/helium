<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test AVT in @include-content-type of xsl:result-document.-->

   <t:template match="/">
      <t:result-document method="html"
                         include-content-type="{doc/foo}"
                         media-type="application/xhtml-xml">
         <html>
            <head>
               <title/>
            </head>
            <body>hello</body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
