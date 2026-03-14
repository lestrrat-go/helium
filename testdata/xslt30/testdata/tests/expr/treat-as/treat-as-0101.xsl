<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schematreatas.example.com/ns/integer"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Simple test 'treat as xs:integer' on a run-time value of type xs:integer or derived by restriction from it. -->

   <xslt:function name="my:temp">
	     <xslt:param name="p1" as="xs:integer"/>
	     <xslt:value-of select="$p1 instance of xs:integer"/>
	     <xslt:value-of select="$p1 instance of my:elem-integer-derived-Type"/>
   </xslt:function>

   <xslt:import-schema namespace="http://www.schematreatas.example.com/ns/integer"
                       schema-location="integerTypesSchemaTreatas.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:value-of select="data(elem-integer-derived) treat as xs:integer"/>
         <xslt:value-of select="my:temp(data(elem-integer-derived) treat as xs:integer)"/>
      </out>
   </xslt:template>
</xslt:transform>
