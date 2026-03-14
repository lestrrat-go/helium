<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:foo="http://test"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs foo">
<!-- Purpose: Test for passing a sequence into a function parameter -->

   <xslt:variable name="values" as="xs:integer*">
      <xslt:sequence select="(1,2,3,4)"/>
   </xslt:variable>

   <xslt:function name="foo:newfunc" as="xs:integer*">
      <xslt:param name="arg1"/>
      <xslt:sequence select="$arg1"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="foo:newfunc($values)" separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
