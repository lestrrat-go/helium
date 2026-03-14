<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://test"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="foo">
<!-- Purpose: Testcase to verify that a static error is raised on a function 
  				declaration with an invalid QName attribute value. -->

   <t:function name="foo:test?">
      <t:param name="param1"/>
      <t:value-of select="$param1"/>
   </t:function>

   <t:template match="/">
      <out>
         <t:value-of select="string('abc')"/>
      </out>
   </t:template>
</t:transform>
