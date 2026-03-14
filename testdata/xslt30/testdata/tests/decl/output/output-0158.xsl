<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" include-content-type="yes"/>
   <!-- Purpose: Test HTML output with include-content-type=yes and <meta http-equiv="Content-Type "> tag already present 
				with additional parameters in the 'content' attribute value. Example from spec.-->

   <t:template match="/">
      <HTML>
         <HEAD>
            <META HTTP-EQUIV="Content-Type" CONTENT="text/html;version='3.0'"></META>
         </HEAD>
         <BODY>
 this should have a meta tag
</BODY>
      </HTML>
   </t:template>
</t:transform>
