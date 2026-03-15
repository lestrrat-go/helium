<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Show that schema-element($name) test will fail if no element is found in the source file 
  				with name matching $name or any of the element names in its substitution group.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                    schema-location="subgroupTypeNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:value-of select="//schema-element(my:book)/my:price"/>
      </out>
   </t:template>
</t:transform>
