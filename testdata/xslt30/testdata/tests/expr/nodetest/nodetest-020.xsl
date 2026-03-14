<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration with a substitution group. 
  				The name of the candidate node matches the name of an element in the substitution group headed by $name. 
  				The type of the candidate node(and the node in the substitutio group) is derived directly from the type 
  				of the declaration in the schema. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                    schema-location="subgroupTypeNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:value-of select="schema-element(my:book)/my:page"/>
      </out>
   </t:template>
</t:transform>
