<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with xsl:template that contains built-in types derived by restriction 
  				from xs:decimal,  @as="xs:float".(type promotion) Types tested:
				- xs:long, xs:nonNegativeInteger, xs:int, xs:byte-->

   <xslt:template match="/">
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
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:float">
      <xslt:value-of select="/doc-schemaas/elem-int"/>
   </xslt:template>

   <xslt:template name="temp2" as="xs:float">
      <xslt:value-of select="/doc-schemaas/elem-nonNegativeInteger"/>
   </xslt:template>

   <xslt:template name="temp3" as="xs:float">
      <xslt:value-of select="/doc-schemaas/elem-long"/>
   </xslt:template>

   <xslt:template name="temp4" as="xs:float">
      <xslt:value-of select="/doc-schemaas/elem-byte"/>
   </xslt:template>
</xslt:transform>
