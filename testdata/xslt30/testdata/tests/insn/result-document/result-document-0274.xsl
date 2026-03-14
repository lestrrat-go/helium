<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Purpose: Test HTML result document with include-content-type="1", the result tree has a 'head' element.  -->

   <t:template match="/">
      <t:result-document method="html" media-type="application/xhtml-xml" include-content-type="1">
         <HTML>
            <HEAD> The head </HEAD>
            <BODY> This should have a META Tag </BODY>
         </HTML>
      </t:result-document>
   </t:template>
</t:transform>
