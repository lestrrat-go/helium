<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" media-type="application/xhtml-xml" indent="no"/>
   <!-- Purpose: Test HTML output with default value for @include-content-type, 
  				the result tree has a 'head' element. (default is 'yes')  -->

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
