<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template which contains a sequence of xs:untypedAtomic values 
  				and @as is a built-in type with occurrence indicator (+). Values for @as tested:
				-all built-in primitive types (except xs:QName), xs:integer, xs:dayTimeDuration,
				xs:yearMonthDuration -->

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
      </out>
   </xslt:template>

   <xslt:template name="a1" as="xs:duration+">
      <xslt:sequence select="/doc/item1"/>
   </xslt:template>

   <xslt:template name="a2" as="xs:dateTime+">
      <xslt:sequence select="/doc/item2"/>
   </xslt:template>

   <xslt:template name="a3" as="xs:time+">
      <xslt:sequence select="/doc/item3"/>
   </xslt:template>

   <xslt:template name="a4" as="xs:date+">
      <xslt:sequence select="/doc/item4"/>
   </xslt:template>

   <xslt:template name="a5" as="xs:gYearMonth+">
      <xslt:sequence select="/doc/item5"/>
   </xslt:template>

   <xslt:template name="a6" as="xs:gYear+">
      <xslt:sequence select="/doc/item6"/>
   </xslt:template>

   <xslt:template name="a7" as="xs:gMonthDay+">
      <xslt:sequence select="/doc/item7"/>
   </xslt:template>

   <xslt:template name="a8" as="xs:gDay+">
      <xslt:sequence select="/doc/item8"/>
   </xslt:template>

   <xslt:template name="a9" as="xs:gMonth+">
      <xslt:sequence select="/doc/item9"/>
   </xslt:template>

   <xslt:template name="a10" as="xs:boolean+">
      <xslt:sequence select="/doc/item10"/>
   </xslt:template>

   <xslt:template name="a11" as="xs:base64Binary+">
      <xslt:sequence select="/doc/item11"/>
   </xslt:template>

   <xslt:template name="a12" as="xs:hexBinary+">
      <xslt:sequence select="/doc/item12"/>
   </xslt:template>

   <xslt:template name="a13" as="xs:float+">
      <xslt:sequence select="/doc/item13"/>
   </xslt:template>

   <xslt:template name="a14" as="xs:decimal+">
      <xslt:sequence select="/doc/item14"/>
   </xslt:template>

   <xslt:template name="a15" as="xs:double+">
      <xslt:sequence select="(/doc/item15[1] ,abs(/doc/item15[2]), /doc/item15[3])"/>
   </xslt:template>

   <xslt:template name="a16" as="xs:integer+">
      <xslt:sequence select="/doc/item16"/>
   </xslt:template>

   <xslt:template name="a17" as="xs:anyURI+">
      <xslt:sequence select="/doc/item17"/>
   </xslt:template>

   <xslt:template name="a18" as="xs:dayTimeDuration+">
      <xslt:sequence select="/doc/item18"/>
   </xslt:template>

   <xslt:template name="a19" as="xs:yearMonthDuration+">
      <xslt:sequence select="/doc/item19"/>
   </xslt:template>

   <xslt:template name="a20" as="xs:string+">
      <xslt:sequence select="/doc/item20"/>
   </xslt:template>
</xslt:transform>
