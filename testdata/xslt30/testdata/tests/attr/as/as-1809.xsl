<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of tunnel xsl:with-param where @select contains user-defined types derived by 
  				restriction from xs:decimal or from xs:float, @as="xs:double".(type promotion) -->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="my:userNode">
            <xslt:with-param name="par1" select="my:userNode/my:de1-decimal-enumeration-inline"
               tunnel="yes" as="xs:double"/>
            <xslt:with-param name="par2" select="my:userNode/my:derived-float" tunnel="yes"
               as="xs:double"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="my:userNode">
      <xslt:apply-templates select="my:elem-anyURI"/>
   </xslt:template>

   <xslt:template match="my:elem-anyURI">
      <xslt:param name="par1" select="1.0" tunnel="yes"/>
      <xslt:param name="par2" select="1.0" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of my:de1-decimal-enumeration-Inline"/>
         <xslt:value-of select="$par1 instance of xs:double"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2"/>
         <xslt:value-of select="$par2 instance of my:derived-float-Type"/>
         <xslt:value-of select="$par2 instance of xs:double"/>
      </par2>
   </xslt:template>
</xslt:transform>
