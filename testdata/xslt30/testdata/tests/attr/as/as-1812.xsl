<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of xsl:template with @as="schema-attribute($QName)". Sequence constructor contains an 
  				attribute from input file or an explicitly created xsl:attribute, with the same type or 
  				typed derived from the one for $QName.-->

   <t:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <t:template match="/">
      <out>
         <temp1>
            <t:call-template name="temp1"/>
         </temp1>
         <temp2>
            <t:call-template name="temp2"/>
         </temp2>
      </out>
   </t:template>

   <t:template name="temp1" as="schema-attribute(my:specialPart)">
      <t:copy-of select="my:userNode/@my:specialPart" validation="strict"/>
   </t:template>

   <t:template name="temp2" as="schema-attribute(my:specialPart)">
      <t:attribute name="my:specialPart" type="my:specialPartNumber">000-AB</t:attribute>
   </t:template>
</t:transform>
