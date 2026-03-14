<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document and @include-content-type="yes", method="text". 
  				@include-content-type should be ignored.-->

   <t:template match="/">
      <t:result-document method="text"
                         include-content-type="yes"
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
