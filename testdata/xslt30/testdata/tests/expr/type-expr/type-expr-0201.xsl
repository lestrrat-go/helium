<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.schemaexpr.example.com/ns/various"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test op:bollean-equal ('eq') where the operands are of type xs:boolean or a type derived by restriction 
  				from it; operands come from input file -->

   <xslt:import-schema namespace="http://www.schemaexpr.example.com/ns/various"
      schema-location="variousTypesSchemaExpr.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <types>
            <xslt:value-of select="data(elem-boolean[1]) instance of xs:boolean"/>
            <xslt:value-of select="data(elem-boolean[2]) instance of xs:boolean"/>
            <xslt:value-of select="data(elem-boolean-derived[1]) instance of xs:boolean"/>
            <xslt:value-of
               select="data(elem-boolean-derived[1]) instance of my:elem-boolean-derived-Type"/>
            <xslt:value-of select="data(elem-boolean-derived[2]) instance of xs:boolean"/>
            <xslt:value-of
               select="data(elem-boolean-derived[2]) instance of my:elem-boolean-derived-Type"/>
         </types>
         <e1>
            <xslt:value-of select="data(elem-boolean[1]) eq data(elem-boolean[1])"/>
         </e1>
         <e2>
            <xslt:value-of select="data(elem-boolean[1]) eq data(elem-boolean[2])"/>
         </e2>
         <e3>
            <xslt:value-of select="data(elem-boolean[1]) eq data(elem-boolean-derived[1])"/>
         </e3>
         <e4>
            <xslt:value-of select="data(elem-boolean[1]) eq data(elem-boolean-derived[2])"/>
         </e4>
         <e5>
            <xslt:value-of select="data(elem-boolean-derived[1]) eq data(elem-boolean-derived[1])"/>
         </e5>
         <e6>
            <xslt:value-of select="data(elem-boolean-derived[2]) eq data(elem-boolean-derived[1])"/>
         </e6>
      </out>
   </xslt:template>
</xslt:transform>
