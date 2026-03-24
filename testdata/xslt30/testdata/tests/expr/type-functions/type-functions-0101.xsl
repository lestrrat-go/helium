<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.schemafunc.example.com/ns/various"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test fn:dateTime with arguments coming from typed nodes in input source. Types tested:
   				- xs:date and xs:time; - user-defined types derived by restriction from xs:date and xs:time -->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
      schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <e1>
            <xslt:value-of select="data(elem-date) instance of xs:date"/>
            <xslt:value-of select="data(elem-time) instance of xs:time"/>
            <xslt:value-of select="data(elem-date-derived) instance of my:elem-date-derived-Type"/>
            <xslt:value-of select="data(elem-time-derived) instance of my:elem-time-derived-Type"/>
            <xslt:value-of select="data(elem-date-derived) instance of xs:date"/>
            <xslt:value-of select="data(elem-time-derived) instance of xs:time"/>
         </e1>
         <e2>
            <xslt:value-of select="dateTime(elem-date, elem-time)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="dateTime(elem-date, elem-time) instance of xs:dateTime"/>
         </e2>
         <e3>
            <xslt:value-of select="dateTime(elem-date-derived, elem-time-derived)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="dateTime(elem-date-derived, elem-time-derived)  instance of xs:dateTime"/>
         </e3>
      </out>
   </xslt:template>
</xslt:transform>
