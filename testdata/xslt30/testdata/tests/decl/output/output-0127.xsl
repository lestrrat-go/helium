<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" include-content-type="no" indent="no"/>
   <!-- Purpose: Test XHTML output with include-content-type=no. 
  				The document element of the final result tree is 
  				'html' in the XHTML namespace. -->

   <t:template match="/">
      <html>
         <head>the head</head>
         <body>this should not have a meta tag</body>
      </html>
   </t:template>
</t:transform>
