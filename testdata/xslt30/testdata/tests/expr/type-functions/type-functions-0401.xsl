<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schemafunc.example.com/ns/various"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test fn:year-from-date with arguments coming from typed nodes in input source. Types tested:
   				- xs:date
   				- user-defined types derived by restriction from xs:date-->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
                       schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:text>
</xslt:text>
         <e1>
            <xslt:value-of select="data(elem-date) instance of xs:date"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="year-from-date(elem-date)"/>
         </e1>
         <xslt:text>
</xslt:text>
         <e2>
            <xslt:value-of select="data(elem-date-derived) instance of xs:date"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-date-derived) instance of my:elem-date-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="year-from-date(elem-date-derived)"/>
         </e2>
      </out>
   </xslt:template>
</xslt:transform>
