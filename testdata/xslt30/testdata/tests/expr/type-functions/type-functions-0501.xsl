<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schemafunc.example.com/ns/various"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test fn:local-name-from-QName with arguments coming from typed nodes in input source. Types tested:
   				- xs:QName
   				- user-defined types derived by restriction from xs:QName-->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
                       schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <e1>
            <xslt:value-of select="data(elem-QName) instance of xs:QName"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="local-name-from-QName(elem-QName)"/>
         </e1>
         <e2>
            <xslt:value-of select="data(elem-QName-derived) instance of xs:QName"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-QName-derived) instance of my:elem-QName-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="local-name-from-QName(elem-QName-derived)"/>
         </e2>
      </out>
   </xslt:template>
</xslt:transform>
