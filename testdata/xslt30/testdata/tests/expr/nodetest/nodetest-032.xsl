<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test schema-attribute($name) with an axis. Axes tested-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
      schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
      <out>
         <E1>
            <t:value-of select="ancestor::schema-attribute(my:count)"/>
         </E1>
         <E2>
            <t:value-of select="ancestor-or-self::schema-attribute(my:colors)"/>
         </E2>
         <E3>
            <t:value-of select="attribute::schema-attribute(my:listParts)"/>
         </E3>
         <E4>
            <t:value-of select="self::schema-attribute(my:specialPart)"/>
         </E4>
         <E5>
            <t:value-of select="child::schema-attribute(my:listParts)"/>
         </E5>
      </out>
   </t:template>
</t:transform>
