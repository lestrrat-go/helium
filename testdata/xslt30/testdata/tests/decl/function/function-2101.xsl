<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://test"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="foo">
<!-- Purpose: Testcase to verify that a static error is raised on a function declaration without name attribute.-->

   <t:function>
      <t:param name="param1"/>
   </t:function>

   <t:template match="/">
      <out>
         <t:value-of select="foo:test()"/>
      </out>
   </t:template>
</t:transform>
