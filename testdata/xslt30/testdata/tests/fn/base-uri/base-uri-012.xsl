<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Testcase with fn:base-uri($arg) where $arg is the empty sequence. 
  				The empty sequence is returned.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:value-of select="base-uri(())"/>
      </out>
   </t:template>
</t:transform>
