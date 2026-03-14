<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/attributes"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Show that schema-attribute($name) test will fail if no attribute is found in the source file with name matching $name.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/attributes"
                    schema-location="attributesNodeTest.xsd"/>

   <t:template match="/my:people">
	     <out>
         <t:apply-templates select="//name"/>
      </out>
   </t:template>

   <t:template match="name">
	     <t:value-of select="schema-attribute(my:nationality)"/>
   </t:template>
</t:transform>
