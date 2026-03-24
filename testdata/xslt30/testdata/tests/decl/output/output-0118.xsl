<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             indent="no"
             doctype-public="-//W3C//DTD HTML 4.0 Transitional"/>
   <!-- Purpose: Processing instructions should be terminated with "?>". Output method is XHTML. -->

   <t:template match="/">
      <html>
         <t:processing-instruction name="my-pi">href="book.css" type="text/css"</t:processing-instruction>
      </html>
   </t:template>
</t:transform>
