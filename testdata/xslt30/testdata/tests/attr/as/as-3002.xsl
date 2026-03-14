<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/union-list" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of xsl:template which contains a typed element node from input file, 
  				or explicitly created xsl:element or LRE with xsl:type, 
  				@as="element(QName, user-defined-list-type)". Item types in the list are:
				- built-in primitive, built-in derived, user-defined atomic, union-->

   <xslt:import-schema namespace="http://www.example.com/ns/union-list"
      schema-location="list-unionSchemaAs.xsd"/>

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
         <temp5>
            <xslt:call-template name="temp5"/>
         </temp5>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="element(my:list-builtin, xs:NMTOKENS)">
      <xslt:copy-of select="my:userNode/my:list-builtin" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp2" as="element(my:list-primitive, my:myPrimitiveListType)">
      <my:list-primitive xslt:validation="strict">2.2 4.00099 -0.1111 7</my:list-primitive>
   </xslt:template>

   <xslt:template name="temp3" as="element(my:list-derived, my:myDerivedListType)">
      <xslt:element name="my:list-derived" type="my:myDerivedListType">EN PT BG</xslt:element>
   </xslt:template>

   <xslt:template name="temp4" as="element(my:list-user-defined, my:myListType)">
      <xslt:copy-of select="my:userNode/my:list-user-defined" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp5" as="element(my:list-union, my:myUnionListType)">
      <my:list-union xslt:type="my:myUnionListType">5 111-GG 188</my:list-union>
   </xslt:template>
</xslt:transform>
