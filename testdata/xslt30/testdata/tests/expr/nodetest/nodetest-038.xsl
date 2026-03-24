<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration of a complex type and 
  				the type of candidate nodes is set by xsi:type. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                    schema-location="subgroupTypeNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:value-of select="element(my:article)/my:page"/>
      </out>
   </t:template>
</t:transform>
