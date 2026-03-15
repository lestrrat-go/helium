<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test dynamic html-version attribute -->

   <t:template match="/">
	     <t:result-document method="xhtml" html-version="{$param}">
	        <html xmlns="http://www.w3.org/1999/xhtml">
	           <head><title>Heading</title></head>
	           <body>
	              <p>Hello, world!</p>
	           </body>
	        </html>
	     </t:result-document>
   </t:template>
   
   <t:param name="param" select="1.0"/>
</t:transform>
