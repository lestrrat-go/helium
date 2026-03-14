<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" indent="no"/>
   <!-- Purpose: Test serialization of invalid XHTML. Not an error. -->

   <t:template match="/">
      <html>
         <h1>Introduction</h1>
Welcome to this document on XHTML.
 </html>
      <h1>Introduction</h1>
Welcome to this document on XHTML.
</t:template>
</t:transform>
