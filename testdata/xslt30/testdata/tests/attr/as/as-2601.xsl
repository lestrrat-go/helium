<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with xsl:template which contains values (explicitly constructed or obtained at run time) of the same type  as 
  				the built-in derived atomic type specified in @as, use occurrence indicator (*). Sequence constructor contains:
				- a sequence of typed values, a single typed value-->

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
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:token*">
      <xslt:value-of select="/doc-schemaas/elem-token"/>
   </xslt:template>

   <xslt:template name="temp2" as="xs:language*">
      <xslt:value-of select="/doc-schemaas/elem-language"/>
   </xslt:template>

   <xslt:template name="temp3" as="xs:int*">
      <xslt:value-of select="/doc-schemaas/elem-int"/>
      <xslt:value-of select="xs:int(2000)"/>
   </xslt:template>

   <xslt:template name="temp4" as="xs:negativeInteger*">
      <xslt:value-of select="xs:negativeInteger(-2)"/>
      <xslt:value-of select="xs:negativeInteger(-9999999)"/>
      <xslt:value-of select="/doc-schemaas/elem-negativeInteger"/>
   </xslt:template>

   <xslt:template name="temp5" as="xs:byte*">
      <xslt:sequence select="/doc-schemaas/elem-byte"/>
   </xslt:template>

   <xslt:template name="temp6" as="xs:ID*">
      <xslt:value-of select="/doc-schemaas/elem-ID-attr/@attr1"/>
   </xslt:template>
</xslt:transform>
