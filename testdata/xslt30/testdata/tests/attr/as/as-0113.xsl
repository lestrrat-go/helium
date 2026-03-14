<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://myexamplefunc.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with xsl:function where where the value in the sequence constructor is explicitly 
  				constructed and of the same type as the built-in atomic type in @as. Types tested are:
				- xs:string, xs:double, xs:integer, xs:dayTimeDuration, xs:QName -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func1" as="xs:double">
	     <xslt:value-of select="xs:double(5.4321E-1001)"/>
   </xslt:function>

   <xslt:function name="my:func2" as="xs:double">
	     <xslt:value-of select="xs:double(-0.0E0)"/>
   </xslt:function>

   <xslt:function name="my:func3" as="xs:double">
	     <xslt:value-of select="xs:double('INF')"/>
   </xslt:function>

   <xslt:function name="my:func4" as="xs:integer">
	     <xslt:value-of select="xs:integer(43)"/>
   </xslt:function>

   <xslt:function name="my:func5" as="xs:integer">
	     <xslt:value-of select="xs:integer(-300)"/>
   </xslt:function>

   <xslt:function name="my:func6" as="xs:anyURI">
	     <xslt:value-of select="xs:QName('my:local')"/>
   </xslt:function>

   <xslt:function name="my:func7" as="xs:dayTimeDuration">
	     <xslt:value-of select="xs:dayTimeDuration('PT99.999S')"/>
   </xslt:function>

   <xslt:function name="my:func8" as="xs:dayTimeDuration">
	     <xslt:value-of select="xs:dayTimeDuration('-PT100M')"/>
   </xslt:function>

   <xslt:function name="my:func9" as="xs:string">
	     <xslt:value-of select="xs:string('hello')"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         
         <func1>
            <xslt:value-of select="my:func1() instance of xs:double"/>
         </func1>
         
         <func2>
            <xslt:value-of select="my:func2() instance of xs:double"/>
         </func2>
         
         <func3>
            <xslt:value-of select="my:func3() instance of xs:double"/>
         </func3>
         
         <func4>
            <xslt:value-of select="my:func4() instance of xs:integer"/>
         </func4>
         
         <func5>
            <xslt:value-of select="my:func5() instance of xs:integer"/>
         </func5>
         
         <func6>
            <xslt:value-of select="my:func6() instance of xs:anyURI"/>
         </func6>
         
         <func7>
            <xslt:value-of select="my:func7() instance of xs:dayTimeDuration"/>
         </func7>
         
         <func8>
            <xslt:value-of select="my:func8() instance of xs:dayTimeDuration"/>
         </func8>
         
         <func9>
            <xslt:value-of select="my:func9() instance of xs:string"/>
         </func9>
      </out>
   </xslt:template>
</xslt:transform>
