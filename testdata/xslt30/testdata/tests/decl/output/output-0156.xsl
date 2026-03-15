<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xml"
             include-content-type="no"
             indent="no"
             encoding="UTF-8"/>
   <!-- Purpose: Test that @include-content-type with XML output method is ignored.-->

   <t:template match="/">
      <out>Hello</out>
   </t:template>
</t:transform>
