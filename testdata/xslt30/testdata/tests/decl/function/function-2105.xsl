<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.bar.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="foo">
<!-- Purpose: Testcase to verify that a static error is raised if xsl:function is 
  				declared within a sequence-constructor (it is not a top level element)-->

   <t:template match="/">
      <t:function name="foo:test">
         <t:param name="param1"/>
         <t:value-of select="$param1"/>
      </t:function>
      <out>
         <t:value-of select="foo:test('abc')"/>
      </out>
   </t:template>
</t:transform>
