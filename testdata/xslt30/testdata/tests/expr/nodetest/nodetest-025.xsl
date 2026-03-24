<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my"
   input-type-annotations="strip" version="2.0">
   <!-- Purpose: Show that stylesheet attribute @input-type-annotation="strip" makes the schema-attribute test fail.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
      schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="my:complexSimpleContentElem">
      <E1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-attribute(my:count)"/>
      </E1>
   </t:template>
   
   <t:template match="/my:userNode">
      <out>
         <t:apply-templates select="my:complexSimpleContentElem|my:onlyAttributes"/>
         <E4>
            <t:value-of select="schema-attribute(my:specialPart)"/>
         </E4>
         <E5>
            <t:value-of select="schema-attribute(my:listParts)"/>
         </E5>
      </out>
   </t:template>

   <t:template match="my:onlyAttributes">
      <E2 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-attribute(my:colors)"/>
      </E2>
      <E3 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-attribute(my:union)"/>
      </E3>
   </t:template>
</t:transform>
