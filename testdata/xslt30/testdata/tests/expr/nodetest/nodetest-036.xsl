<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test schema-element($name) with a typed element created inside a variable.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
      schema-location="variousTypesNodeTest.xsd"/>
   
   <t:template match="/">
      <out>
         <t:apply-templates select="$var" />
      </out>
   </t:template>
   
   <t:template match="*">
      <E1><t:value-of select="schema-element(my:complexSimpleContentElem)" /></E1>
   </t:template>
   
   <t:variable name="var" as="schema-element(my:userNode)">
      <my:userNode t:validation="strict">
         <my:complexSimpleContentElem my:count="ten" >
            10
         </my:complexSimpleContentElem>
      </my:userNode>
   </t:variable>
   
</t:transform>
