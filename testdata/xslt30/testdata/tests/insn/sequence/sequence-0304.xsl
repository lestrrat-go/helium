<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:foo="http://test"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs foo">
<!-- Purpose: Test for xsl:sequence in a function passed from a param -->

   <xslt:function name="foo:newfunc" as="xs:string*">
      <xslt:param name="arg1" as="xs:string*"/>
      <xslt:sequence select="$arg1"/>
   </xslt:function>

   <xslt:variable name="values" as="xs:string*">
      <xslt:sequence select="('a','b','c')"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="foo:newfunc($values)" separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
