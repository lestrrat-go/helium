<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test that html-version attribute is accepted -->

   <t:template match="/">
	     <t:result-document method="xhtml" html-version="five">
	        <html xmlns="http://www.w3.org/1999/xhtml">
	           <head><title>Heading</title></head>
	           <body>
	              <p>Hello, world!</p>
	           </body>
	        </html>
	     </t:result-document>
   </t:template>
</t:transform>
