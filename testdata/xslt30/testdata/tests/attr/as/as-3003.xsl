<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/union-list" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of tunnel xsl:with-param with @select selecting a typed element node 
  				from input file, @as="element(*, user-defined-list-type)". Item types in the list are:
				- built-in primitive, built-in derived, user-defined atomic, union-->

   <xslt:import-schema namespace="http://www.example.com/ns/union-list"
      schema-location="list-unionSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
            <xslt:with-param name="par1" select="/my:userNode/my:list-builtin"
               as="element(*, xs:NMTOKENS)" tunnel="yes"/>
            <xslt:with-param name="par2" select="/my:userNode/my:list-primitive"
               as="element(*, my:myPrimitiveListType)" tunnel="yes"/>
            <xslt:with-param name="par3" select="/my:userNode/my:list-derived"
               as="element(*, my:myDerivedListType)" tunnel="yes"/>
            <xslt:with-param name="par4" select="/my:userNode/my:list-user-defined"
               as="element(*, my:myListType)" tunnel="yes"/>
            <xslt:with-param name="par5" select="/my:userNode/my:list-union"
               as="element(*, my:myUnionListType)" tunnel="yes"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="my:userNode">
      <xslt:apply-templates select="my:list-builtin"/>
   </xslt:template>

   <xslt:template match="my:list-builtin">
      <xslt:param name="par1" select="'1.0'" tunnel="yes"/>
      <xslt:param name="par2" select="'1.0'" tunnel="yes"/>
      <xslt:param name="par3" select="'1.0'" tunnel="yes"/>
      <xslt:param name="par4" select="'1.0'" tunnel="yes"/>
      <xslt:param name="par5" select="'1.0'" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1 instance of element(*, xs:NMTOKENS)"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2 instance of element(*, my:myPrimitiveListType)"/>
      </par2>
      <par3>
         <xslt:value-of select="$par3 instance of element(*, my:myDerivedListType)"/>
      </par3>
      <par4>
         <xslt:value-of select="$par4 instance of element(*, my:myListType)"/>
      </par4>
      <par5>
         <xslt:value-of select="$par5 instance of element(*, my:myUnionListType)"/>
      </par5>
   </xslt:template>
</xslt:transform>
