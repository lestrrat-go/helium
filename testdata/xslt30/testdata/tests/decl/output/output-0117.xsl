<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             indent="no"
             doctype-public="-//W3C//DTD HTML 4.0 Transitional"/>
   <!-- Purpose: Boolean attributes should not be output in minimized form. -->

   <t:template match="/">
      <html>
         <Option selected="selected"/>
      </html>
   </t:template>
</t:transform>
