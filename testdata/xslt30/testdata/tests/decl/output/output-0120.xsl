<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             cdata-section-elements="test"
             encoding="UTF-8"
             indent="no"/>
   <!-- Purpose: Test of cdata-section-elements processing. <example>
  	   should not be processed. -->

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <example>&gt;&gt;&gt;SHOULD NOT BE WRAPPED WITH cdata section&lt;&lt;&lt;</example>
         <t:text>
</t:text>
         <test>&gt;&gt;&gt;SHOULD BE WRAPPED WITH cdata section&lt;&lt;&lt;</test>
      </out>
   </t:template>
</t:transform>
