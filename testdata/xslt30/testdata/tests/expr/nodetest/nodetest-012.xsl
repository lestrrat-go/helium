<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Use xsl:apply-templates with schema-element() test. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
	     <out>
         <t:apply-templates select="schema-element(my:complexMixedUserElem)"/>
      </out>
   </t:template>

   <t:template match="my:complexMixedUserElem">
	     <t:value-of select="*[1]"/>
   </t:template>
</t:transform>
