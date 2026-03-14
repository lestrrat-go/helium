<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://myexamplefunc.org"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:function where the value in the sequence constructor is explicitly 
  				constructed and is of the same type as the built-in derived atomic type in @as. 
  				Types tested are: -xs:nonPositiveInteger, xs:int, xs:long, xs:unsignedByte,
  				xs:NMTOKEN, xs:ID. Verify the result of the function is of the type specified in @as. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func1" as="xs:nonPositiveInteger">
      <xslt:value-of select="xs:nonPositiveInteger(-7632460)"/>
   </xslt:function>

   <xslt:function name="my:func2" as="xs:int">
      <xslt:value-of select="xs:int(131072)"/>
   </xslt:function>

   <xslt:function name="my:func3" as="xs:long">
      <xslt:value-of select="xs:long(-9223372)"/>
   </xslt:function>

   <xslt:function name="my:func4" as="xs:unsignedByte">
      <xslt:value-of select="xs:unsignedByte(110)"/>
   </xslt:function>

   <xslt:function name="my:func5" as="xs:NMTOKEN">
      <xslt:value-of select="xs:NMTOKEN('business-enforcement::link-chain.common')"/>
   </xslt:function>

   <xslt:function name="my:func6" as="xs:ID">
      <xslt:value-of select="xs:ID('_a')"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <func1>
            <xslt:value-of select="my:func1() instance of xs:nonPositiveInteger"/>
         </func1>
         <func2>
            <xslt:value-of select="my:func2() instance of xs:int"/>
         </func2>
         <func3>
            <xslt:value-of select="my:func3() instance of xs:long"/>
         </func3>
         <func4>
            <xslt:value-of select="my:func4() instance of xs:unsignedByte"/>
         </func4>
         <func5>
            <xslt:value-of select="my:func5() instance of xs:NMTOKEN"/>
         </func5>
         <func6>
            <xslt:value-of select="my:func6() instance of xs:ID"/>
         </func6>
      </out>
   </xslt:template>
</xslt:transform>
