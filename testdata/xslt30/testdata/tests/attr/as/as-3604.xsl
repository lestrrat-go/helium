<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0"
   xmlns:my="http://www.example.com/ns/attr"
   xmlns:xs="http://www.w3.org/2001/XMLSchema">
<!-- Purpose: As -3602, but attribute is invalid -->
   
   <t:import-schema namespace="http://www.example.com/ns/attr"
      schema-location="attributesSchemaAs.xsd"/>

   <t:template match="/my:people">
      
      <t:param name="par1" select="string((//name)[1])" as="xs:string"/>

      <t:param name="par2" as="attribute()">
         <t:attribute name="my:name-nationality" validation="strict">patagonian</t:attribute>
      </t:param>

      <out>
         <t:value-of select="$par1 instance of attribute()"/>
         <t:value-of select="$par2 instance of schema-attribute(my:name-nationality)"/>
      </out>
   </t:template>
</t:transform>
