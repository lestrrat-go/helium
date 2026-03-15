<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <t:output method="html"
             media-type="application/xhtml-xml"
             include-content-type=" 1 "/>
   <!-- Purpose: Test HTML output with include-content-type=" 1 ", the result tree has a 'head' element.  -->

   <t:template match="/">
      <HTML>
         <HEAD>
  The head
</HEAD>
         <BODY>
 This should have a META Tag
</BODY>
      </HTML>
   </t:template>
</t:transform>
