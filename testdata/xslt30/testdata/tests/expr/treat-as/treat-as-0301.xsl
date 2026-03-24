<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schematreatas.example.com/ns/integer"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with 'treat as' where the value is casted to the required SequenceType at run time.-->

   <xslt:import-schema namespace="http://www.schematreatas.example.com/ns/integer"
                       schema-location="integerTypesSchemaTreatas.xsd"/>

   <xslt:function name="my:temp">
	     <xslt:param name="p1" as="my:elem-integer-derived-Type"/>
	     <xslt:value-of select="$p1 instance of xs:integer"/>
	     <xslt:value-of select="$p1 instance of my:elem-integer-derived-Type"/>
   </xslt:function>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:value-of select="(data(elem-integer) cast as my:elem-integer-derived-Type) treat as my:elem-integer-derived-Type"/>
         <xslt:value-of select="my:temp(data(elem-integer cast as my:elem-integer-derived-Type) treat as my:elem-integer-derived-Type)"/>
      </out>
   </xslt:template>
</xslt:transform>
