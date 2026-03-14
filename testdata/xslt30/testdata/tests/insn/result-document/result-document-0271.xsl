<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: Test HTML result document with include-content-type value set to " no ".  -->

   <t:template match="/">
      <t:result-document method="html" include-content-type=" no ">
         <HTML>
            <HEAD> The head </HEAD>
            <BODY> This should not have a META Tag </BODY>
         </HTML>
      </t:result-document>
   </t:template>
</t:transform>
