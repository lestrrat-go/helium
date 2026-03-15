<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" include-content-type="yes"/>
   <!-- Purpose: Test HTML output with include-content-type=yes and <meta http-equiv="Content-Type "> tag already present.-->

   <t:template match="/">
      <HTML>
         <HEAD>
            <META http-equiv="Content-Type" content="text/html; charset=UTF-16"/>
         </HEAD>
         <BODY>
 This should have a META Tag
</BODY>
      </HTML>
   </t:template>
</t:transform>
