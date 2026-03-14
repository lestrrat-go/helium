<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schemafunc.example.com/ns/various"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test constructor function on user-defined type-->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
                       schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <xslt:variable name="v" select="my:partNumberType('123-XY')"/>
      <out value="{$v}" ok="{$v instance of my:partNumberType}"/>
   </xslt:template>
</xslt:transform>
