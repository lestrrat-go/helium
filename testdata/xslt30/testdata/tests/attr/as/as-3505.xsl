<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of non-tunnel xsl:with-param with  @as="element(*, user-defined-atomic-type) *". @select has one of:
				-(), an empty sequence obtained at run time, a typed element node from input file, a set of typed element 
				nodes from input file-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
            <xslt:with-param name="par1" select="()" as="element(*,my:partNumberType)*" tunnel="no"/>
            <xslt:with-param name="par2" select="/my:user" as="element(*,my:partNumberType)*"
               tunnel="no"/>
            <xslt:with-param name="par3" select="/my:userNode/my:simpleUserElem[1]"
               as="element(*,my:partNumberType)*" tunnel="no"/>
            <xslt:with-param name="par4" select="/my:userNode/my:simpleUserElem"
               as="element(*,my:partNumberType)*" tunnel="no"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="my:userNode">
      <xslt:param name="par1" select="'1.0'"/>
      <xslt:param name="par2" select="'1.0'"/>
      <xslt:param name="par3" select="'1.0'"/>
      <xslt:param name="par4" select="'1.0'"/>
      <par1>
         <xslt:value-of select="$par1 instance of element(*,my:partNumberType)*"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2 instance of element(*,my:partNumberType)*"/>
      </par2>
      <par3>
         <xslt:value-of select="$par3 instance of element(*,my:partNumberType)*"/>
      </par3>
      <par4>
         <xslt:value-of select="$par4 instance of element(*,my:partNumberType)*"/>
      </par4>
   </xslt:template>
</xslt:transform>
