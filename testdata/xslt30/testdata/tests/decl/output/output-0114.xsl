<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             cdata-section-elements="example"
             indent="no"
             encoding="UTF-8"/>
   <!-- Purpose: Test XHTML output with cdata-section-elements that have ]]-->

   <t:template match="/">
      <html>
         <out>
            <example>]]</example>
         </out>
      </html>
   </t:template>
</t:transform>
