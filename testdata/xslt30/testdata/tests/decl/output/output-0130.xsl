<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test with no output method explicitly specified and the document 
  				element of the implicit result tree is html in the XHTML namespace. 
  				Show that output method defaults to XHTML. -->

   <t:template match="/">
      <html>
         <head>
            <title>Default output method</title>
         </head>
         <body>
            <p>Verify<a href="http://exampleÇ.org/Â"> this is XHTML</a>.</p>
         </body>
      </html>
   </t:template>
</t:transform>
