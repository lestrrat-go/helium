<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             media-type="application/postscript"
             indent="no"
             encoding="UTF-8"/>
   <!-- Purpose: Test that @media-type can be specified on output method xhtml. 
  				Value used is "application/postscript" -->

   <t:template match="/">
      <html>
         <head>
  the head
</head>
         <body>
 this should have a meta tag
</body>
      </html>
   </t:template>
</t:transform>
