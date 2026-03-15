<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/attributes"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Show that if the name of a node in the source file complies to the schema-attribute($name) test but its type 
  				(different than xs:untypedAtomic) does not, the test will fail. The candidate node is validated against a local 
  				attribute declaration with the same expanded-QName as the top-level element declaration.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/attributes"
                    schema-location="attributesNodeTest.xsd"/>

   <t:template match="/my:people">
	     <out>
         <t:apply-templates select="//age"/>
      </out>
   </t:template>

   <t:template match="age">
	     <t:value-of select="schema-attribute(my:nationality)"/>
   </t:template>
</t:transform>
