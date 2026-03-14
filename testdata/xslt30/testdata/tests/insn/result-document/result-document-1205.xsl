<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Purpose: Test AVT in @include-content-type of xsl:result-document, which evaluates to " true ".  -->

   <t:template match="/">
      <t:result-document method="html" media-type="application/xhtml-xml" include-content-type="{doc/foo}">
         <HTML>
            <HEAD> The head </HEAD>
            <BODY> This should have a META Tag </BODY>
         </HTML>
      </t:result-document>
   </t:template>
</t:transform>
