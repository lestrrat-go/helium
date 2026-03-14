<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name has explicit prefix and is a top level element declaration 
  				in a schema with target namespace. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:value-of select="schema-element(my:elem-anyURI)"/>
      </out>
   </t:template>
</t:transform>
