<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.schemaexpr.example.com/ns/various"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test op:numeric-add where one operand is xs:decimal and the other is typed numeric value, 
  				either built-in or a user type derived by restriction from built-in numeric.  -->

   <xslt:import-schema namespace="http://www.schemaexpr.example.com/ns/various"
      schema-location="variousTypesSchemaExpr.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:for-each select="*">
            <xslt:element name="e{position()}">
               <xslt:value-of select="data(/my:userNode/*[1])"/>
               <xslt:text> + </xslt:text>
               <xslt:value-of select="data(.)"/>
               <xslt:text> = </xslt:text>
               <xslt:value-of select="data(/my:userNode/*[1]) + data(.)"/>
            </xslt:element>
         </xslt:for-each>
         <xslt:for-each select="*">
            <xslt:element name="f{position()}">
               <xslt:value-of select="data(.)"/>
               <xslt:text> + </xslt:text>
               <xslt:value-of select="data(/my:userNode/*[1])"/>
               <xslt:text> = </xslt:text>
               <xslt:value-of select="data(.) + data(/my:userNode/*[1])"/>
            </xslt:element>
         </xslt:for-each>
      </out>
   </xslt:template>
</xslt:transform>
