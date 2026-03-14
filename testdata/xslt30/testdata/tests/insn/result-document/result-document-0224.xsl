<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document and @include-content-type="yes", method="html", @media-type is set 
  				and there is a &lt;head&gt; and &lt;meta http-equiv="Content-Type"&gt; element in the final result tree.-->

   <t:template match="/">
      <t:result-document method="html"
                         include-content-type="yes"
                         media-type="application/xhtml-xml">
         <html>
            <head>
               <meta http-equiv="Content-Type" content="text/html;version='3.0'"/>
               <title/>
            </head>
            <body>hello</body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
