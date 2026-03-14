<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test of xsl:template with a sequence constructor of LREs and @as= built-in derived atomic type. 
  				Value of LRE is in the lexical space of the type in @as.-->

   <xslt:template match="/doc-schemaas">
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
      <xslt:value-of select="/doc-schemaas/elem-normalizedString"/>
   </xslt:template>

   <xslt:template name="temp2" as="xs:token">
      <xslt:value-of select="/doc-schemaas/elem-token"/>
   </xslt:template>

   <xslt:template name="temp3" as="xs:language">
      <xslt:value-of select="/doc-schemaas/elem-language"/>
   </xslt:template>

   <xslt:template name="temp4" as="xs:Name">
      <xslt:value-of select="/doc-schemaas/elem-Name"/>
   </xslt:template>

   <xslt:template name="temp5" as="xs:NCName">
      <xslt:value-of select="/doc-schemaas/elem-NCName"/>
   </xslt:template>

   <xslt:template name="temp6" as="xs:NMTOKEN">
      <xslt:value-of select="/doc-schemaas/elem-NMTOKEN"/>
   </xslt:template>

   <xslt:template name="temp7" as="xs:int">
      <xslt:value-of select="/doc-schemaas/elem-int"/>
   </xslt:template>

   <xslt:template name="temp8" as="xs:nonPositiveInteger">
      <xslt:value-of select="/doc-schemaas/elem-nonPositiveInteger"/>
   </xslt:template>

   <xslt:template name="temp9" as="xs:positiveInteger">
      <xslt:value-of select="/doc-schemaas/elem-positiveInteger"/>
   </xslt:template>

   <xslt:template name="temp10" as="xs:nonNegativeInteger">
      <xslt:value-of select="/doc-schemaas/elem-nonNegativeInteger"/>
   </xslt:template>

   <xslt:template name="temp11" as="xs:negativeInteger">
      <xslt:value-of select="/doc-schemaas/elem-negativeInteger"/>
   </xslt:template>

   <xslt:template name="temp12" as="xs:long">
      <xslt:value-of select="/doc-schemaas/elem-long"/>
   </xslt:template>

   <xslt:template name="temp13" as="xs:unsignedLong">
      <xslt:value-of select="/doc-schemaas/elem-unsignedLong"/>
   </xslt:template>

   <xslt:template name="temp14" as="xs:unsignedInt">
      <xslt:value-of select="/doc-schemaas/elem-unsignedInt"/>
   </xslt:template>

   <xslt:template name="temp15" as="xs:short">
      <xslt:value-of select="/doc-schemaas/elem-short"/>
   </xslt:template>

   <xslt:template name="temp16" as="xs:unsignedShort">
      <xslt:value-of select="/doc-schemaas/elem-unsignedShort"/>
   </xslt:template>

   <xslt:template name="temp17" as="xs:byte">
      <xslt:value-of select="/doc-schemaas/elem-byte"/>
   </xslt:template>

   <xslt:template name="temp18" as="xs:unsignedByte">
      <xslt:value-of select="/doc-schemaas/elem-unsignedByte"/>
   </xslt:template>

   <xslt:template name="temp19" as="xs:ID">
      <xslt:value-of select="/doc-schemaas/elem-ID-attr/@attr1"/>
   </xslt:template>

   <xslt:template name="temp20" as="xs:IDREF">
      <xslt:value-of select="/doc-schemaas/elem-IDREF-attr/@attr1"/>
   </xslt:template>
</xslt:transform>
