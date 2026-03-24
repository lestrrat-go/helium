<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/attributes"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-attribute($name) where $name is a top level attribute declaration. The type 
  				of the candidate node is derived directly from the type of the declaration in the schema. The 
  				candidate node is validated against a local attribute declaration with the same expanded-QName 
  				as the top-level attribute declation.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/attributes"
                    schema-location="attributesNodeTest.xsd"/>

   <t:template match="/my:people">
	     <out>
         <t:apply-templates/>
      </out>
   </t:template>

   <t:template match="person">
	     <t:value-of select="schema-attribute(my:nationality)"/>
   </t:template>

   <t:template match="text()"/>
</t:transform>
