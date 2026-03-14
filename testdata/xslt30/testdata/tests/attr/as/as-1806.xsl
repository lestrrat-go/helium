<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains user-defined types derived by 
  				restriction from xs:decimal or from xs:float,  @as="xs:double".(type promotion) -->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:userNode/my:de1-decimal-enumeration-inline" as="xs:double"/>

   <xslt:variable name="var2" select="/my:userNode/my:derived-float" as="xs:double"/>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:de1-decimal-enumeration-Inline"/>
            <xslt:value-of select="$var1 instance of xs:double"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of my:derived-float-Type"/>
            <xslt:value-of select="$var2 instance of xs:double"/>
         </var2>
      </out>
   </xslt:template>
</xslt:transform>
