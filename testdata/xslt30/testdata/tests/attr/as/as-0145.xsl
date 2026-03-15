<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of xsl:template with a sequence constructor of LREs and @as= built-in derived atomic type. 
  				Value of LRE is in the lexical space of the type in @as.-->

   <xslt:template match="/doc">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="temp2"/>
         </temp2>
         <temp3>
            <xslt:call-template name="temp3"/>
         </temp3>
         <temp4>
            <xslt:call-template name="temp4"/>
         </temp4>
         <temp5>
            <xslt:call-template name="temp5"/>
         </temp5>
         <temp6>
            <xslt:call-template name="temp6"/>
         </temp6>
         <temp7>
            <xslt:call-template name="temp7"/>
         </temp7>
         <temp8>
            <xslt:call-template name="temp8"/>
         </temp8>
         <temp9>
            <xslt:call-template name="temp9"/>
         </temp9>
         <temp10>
            <xslt:call-template name="temp10"/>
         </temp10>
         <temp11>
            <xslt:call-template name="temp11"/>
         </temp11>
         <temp12>
            <xslt:call-template name="temp12"/>
         </temp12>
         <temp13>
            <xslt:call-template name="temp13"/>
         </temp13>
         <temp14>
            <xslt:call-template name="temp14"/>
         </temp14>
         <temp15>
            <xslt:call-template name="temp15"/>
         </temp15>
         <temp16>
            <xslt:call-template name="temp16"/>
         </temp16>
         <temp17>
            <xslt:call-template name="temp17"/>
         </temp17>
         <temp18>
            <xslt:call-template name="temp18"/>
         </temp18>
         <temp19>
            <xslt:call-template name="temp19"/>
         </temp19>
         <temp20>
            <xslt:call-template name="temp20"/>
         </temp20>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:normalizedString">
      <item>like</item>
   </xslt:template>

   <xslt:template name="temp2" as="xs:token">
      <item>two organizations</item>
   </xslt:template>

   <xslt:template name="temp3" as="xs:language">
      <item>PT</item>
   </xslt:template>

   <xslt:template name="temp4" as="xs:Name">
      <item>a:_underscore_</item>
   </xslt:template>

   <xslt:template name="temp5" as="xs:NCName">
      <item>vwxyz12</item>
   </xslt:template>

   <xslt:template name="temp6" as="xs:NMTOKEN">
      <item>business-enforcement::link-chain.common</item>
   </xslt:template>

   <xslt:template name="temp7" as="xs:int">
      <item>131072</item>
   </xslt:template>

   <xslt:template name="temp8" as="xs:nonPositiveInteger">
      <item>-7632460</item>
   </xslt:template>

   <xslt:template name="temp9" as="xs:positiveInteger">
      <item>18446744</item>
   </xslt:template>

   <xslt:template name="temp10" as="xs:nonNegativeInteger">
      <item>4444</item>
   </xslt:template>

   <xslt:template name="temp11" as="xs:negativeInteger">
      <item>-74146</item>
   </xslt:template>

   <xslt:template name="temp12" as="xs:long">
      <item>-9223372</item>
   </xslt:template>

   <xslt:template name="temp13" as="xs:unsignedLong">
      <item>1844674</item>
   </xslt:template>

   <xslt:template name="temp14" as="xs:unsignedInt">
      <item>573231403</item>
   </xslt:template>

   <xslt:template name="temp15" as="xs:short">
      <item>-32768</item>
   </xslt:template>

   <xslt:template name="temp16" as="xs:unsignedShort">
      <item>65535</item>
   </xslt:template>

   <xslt:template name="temp17" as="xs:byte">
      <item>-128</item>
   </xslt:template>

   <xslt:template name="temp18" as="xs:unsignedByte">
      <item>110</item>
   </xslt:template>

   <xslt:template name="temp19" as="xs:ID">
      <item>_a</item>
   </xslt:template>

   <xslt:template name="temp20" as="xs:IDREF">
      <item>_a</item>
   </xslt:template>
</xslt:transform>
