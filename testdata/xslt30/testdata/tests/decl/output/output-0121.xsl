<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" indent="no"/>
   <!-- Purpose: Test that the default value of @omit-xml-declaration with XHTML output method is 'no'. -->

   <t:template match="/">
      <example>SHOULD have XML Declaration</example>
   </t:template>
</t:transform>
