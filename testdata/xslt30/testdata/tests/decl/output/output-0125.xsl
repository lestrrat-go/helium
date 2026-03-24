<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" include-content-type="no"/>
   <!-- Purpose: Test HTML output with include-content-type value set to "no".  -->

   <t:template match="/">
      <HTML>
         <HEAD>
  The head
</HEAD>
         <BODY>
 This should not have a META Tag
</BODY>
      </HTML>
   </t:template>
</t:transform>
