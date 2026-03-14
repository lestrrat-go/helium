<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration with a substitution group. 
  				The name of the candidate node matches the name of an element in the substitution group headed by $name. 
  				The types of the top-level element declaration and the element in the substritution group are the same.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                    schema-location="subgroupTypeNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:value-of select="schema-element(my:book)/my:price"/>
      </out>
   </t:template>
</t:transform>
