<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with @output-version="5.0" and @method="text" on xsl:result-document. 
  				Version attribute should be ignored.-->

   <t:template match="/">
	     <t:result-document method="text" output-version="5.0">
		       <html>
            <head>
               <title/>
            </head>
            <body>hello</body>
         </html>
	     </t:result-document>
   </t:template>
</t:transform>
