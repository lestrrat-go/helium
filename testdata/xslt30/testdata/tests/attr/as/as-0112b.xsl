<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template which returns an explicitly constructed built-in atomic type 
  				value and @as speficies the same type. Verify no typing error is raised. Value types tested:
				- all built-in primitive types, xs:integer, xs:dayTimeDuration, xs:yearMonthDuration   -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <out>
         <template1>
            <xslt:call-template name="a1"/>
         </template1>
         <template2>
            <xslt:call-template name="a2"/>
         </template2>
         <template3>
            <xslt:call-template name="a3"/>
         </template3>
         <template4>
            <xslt:call-template name="a4"/>
         </template4>
         <template5>
            <xslt:call-template name="a5"/>
         </template5>
         <template6>
            <xslt:call-template name="a6"/>
         </template6>
         <template7>
            <xslt:call-template name="a7"/>
         </template7>
         <template8>
            <xslt:call-template name="a8"/>
         </template8>
         <template9>
            <xslt:call-template name="a9"/>
         </template9>
         <template10>
            <xslt:call-template name="a10"/>
         </template10>
         <template11>
            <xslt:call-template name="a11"/>
         </template11>
         <template12>
            <xslt:call-template name="a12"/>
         </template12>
         <template13>
            <xslt:call-template name="a13"/>
         </template13>
         <template14>
            <xslt:call-template name="a14"/>
         </template14>
         <template15>
            <xslt:call-template name="a15"/>
         </template15>
         <template16>
            <xslt:call-template name="a16"/>
         </template16>
         <template17>
            <xslt:call-template name="a17"/>
         </template17>
         <template18>
            <xslt:call-template name="a18"/>
         </template18>
         <template19>
            <xslt:call-template name="a19"/>
         </template19>
         <template20>
            <xslt:call-template name="a20"/>
         </template20>
         <template21>
            <xslt:call-template name="a21"/>
         </template21>
         <template22>
            <xslt:call-template name="a22"/>
         </template22>
         <template23>
            <xslt:call-template name="a23"/>
         </template23>
         <template24>
            <xslt:call-template name="a24"/>
         </template24>
         <template25>
            <xslt:call-template name="a25"/>
         </template25>
         <template26>
            <xslt:call-template name="a26"/>
         </template26>
         <template27>
            <xslt:call-template name="a27"/>
         </template27>
         <template28>
            <xslt:call-template name="a28"/>
         </template28>
         <template29>
            <xslt:call-template name="a29"/>
         </template29>
         <template30>
            <xslt:call-template name="a30"/>
         </template30>
         <template31>
            <xslt:call-template name="a31"/>
         </template31>
         <template32>
            <xslt:call-template name="a32"/>
         </template32>
         <template34>
            <xslt:call-template name="a34"/>
         </template34>
         <template35>
            <xslt:call-template name="a35"/>
         </template35>
         <template36>
            <xslt:call-template name="a36"/>
         </template36>
         <template37>
            <xslt:call-template name="a37"/>
         </template37>
         <template38>
            <xslt:call-template name="a38"/>
         </template38>
      </out>
   </xslt:template>

   <xslt:template name="a1" as="xs:duration">
      <xslt:value-of select="xs:duration('P1Y2M3DT10H30M23S')"/>
   </xslt:template>

   <xslt:template name="a2" as="xs:duration">
      <xslt:value-of select="xs:duration('-P12M23DT0M59.123S')"/>
   </xslt:template>

   <xslt:template name="a3" as="xs:dateTime">
      <xslt:value-of select="xs:dateTime('2000-01-16T00:00:00Z')"/>
   </xslt:template>

   <xslt:template name="a4" as="xs:dateTime">
      <xslt:value-of select="xs:dateTime('9999-05-31T13:20:00+14:00')"/>
   </xslt:template>

   <xslt:template name="a5" as="xs:time">
      <xslt:value-of select="xs:time('13:20:00-05:00')"/>
   </xslt:template>

   <xslt:template name="a6" as="xs:time">
      <xslt:value-of select="xs:time('00:00:00')"/>
   </xslt:template>

   <xslt:template name="a7" as="xs:date">
      <xslt:value-of select="xs:date('0012-12-03-05:00')"/>
   </xslt:template>

   <xslt:template name="a8" as="xs:date">
      <xslt:value-of select="xs:date('1999-05-31')"/>
   </xslt:template>

   <xslt:template name="a9" as="xs:gYearMonth">
      <xslt:value-of select="xs:gYearMonth('6999-05+14:00')"/>
   </xslt:template>

   <xslt:template name="a10" as="xs:gYearMonth">
      <xslt:value-of select="xs:gYearMonth('1999-05')"/>
   </xslt:template>

   <xslt:template name="a11" as="xs:gYear">
      <xslt:value-of select="xs:gYear('0012-05:00')"/>
   </xslt:template>

   <xslt:template name="a12" as="xs:gYear">
      <xslt:value-of select="xs:gYear('2005')"/>
   </xslt:template>

   <xslt:template name="a13" as="xs:gMonthDay">
      <xslt:value-of select="xs:gMonthDay('--05-31')"/>
   </xslt:template>

   <xslt:template name="a14" as="xs:gMonthDay">
      <xslt:value-of select="xs:gMonthDay('--05-31+14:00')"/>
   </xslt:template>

   <xslt:template name="a15" as="xs:gDay">
      <xslt:value-of select="xs:gDay('---31+14:00')"/>
   </xslt:template>

   <xslt:template name="a16" as="xs:gDay">
      <xslt:value-of select="xs:gDay('---31Z')"/>
   </xslt:template>

   <xslt:template name="a17" as="xs:gMonth">
      <xslt:value-of select="xs:gMonth('--05+14:00')"/>
   </xslt:template>

   <xslt:template name="a18" as="xs:gMonth">
      <xslt:value-of select="xs:gMonth('--05')"/>
   </xslt:template>

   <xslt:template name="a19" as="xs:boolean">
      <xslt:value-of select="xs:boolean('true')"/>
   </xslt:template>

   <xslt:template name="a20" as="xs:base64Binary">
      <xslt:value-of select="xs:base64Binary('PM24')"/>
   </xslt:template>

   <xslt:template name="a21" as="xs:hexBinary">
      <xslt:value-of select="xs:hexBinary('07FF')"/>
   </xslt:template>

   <xslt:template name="a22" as="xs:float">
      <xslt:value-of select="xs:float(1.1234E99)"/>
   </xslt:template>

   <xslt:template name="a23" as="xs:float">
      <xslt:value-of select="xs:float(-1.75e-3)"/>
   </xslt:template>

   <xslt:template name="a24" as="xs:decimal">
      <xslt:value-of select="xs:decimal(-3.420100)"/>
   </xslt:template>

   <xslt:template name="a25" as="xs:decimal">
      <xslt:value-of select="xs:decimal(7.8899)"/>
   </xslt:template>

   <xslt:template name="a26" as="xs:decimal">
      <xslt:value-of select="xs:decimal(0.0)"/>
   </xslt:template>

   <xslt:template name="a27" as="xs:double">
      <xslt:value-of select="abs(xs:double(-0.0E0))"/>
   </xslt:template>

   <xslt:template name="a28" as="xs:double">
      <xslt:value-of select="xs:double(5.4321E-1001)"/>
   </xslt:template>

   <xslt:template name="a29" as="xs:double">
      <xslt:value-of select="xs:double('INF')"/>
   </xslt:template>

   <xslt:template name="a30" as="xs:integer">
      <xslt:value-of select="xs:integer(43)"/>
   </xslt:template>

   <xslt:template name="a31" as="xs:integer">
      <xslt:value-of select="xs:integer(-300)"/>
   </xslt:template>

   <xslt:template name="a32" as="xs:anyURI">
      <xslt:value-of select="xs:anyURI('http://www.mytypetest.org')"/>
   </xslt:template>

   <xslt:template name="a34" as="xs:dayTimeDuration">
      <xslt:value-of select="xs:dayTimeDuration('PT99.999S')"/>
   </xslt:template>

   <xslt:template name="a35" as="xs:dayTimeDuration">
      <xslt:value-of select="xs:dayTimeDuration('-PT100M')"/>
   </xslt:template>

   <xslt:template name="a36" as="xs:yearMonthDuration">
      <xslt:value-of select="xs:yearMonthDuration('P1Y2M')"/>
   </xslt:template>

   <xslt:template name="a37" as="xs:yearMonthDuration">
      <xslt:value-of select="xs:yearMonthDuration('-P21M')"/>
   </xslt:template>

   <xslt:template name="a38" as="xs:string">
      <xslt:value-of select="xs:string('hello')"/>
   </xslt:template>
</xslt:transform>
