<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration 
  				in a schema with no target namespace.-->

   <t:import-schema schema-location="builtinTypeNodeTest.xsd"/>

   <t:template match="/doc">
	     <out>
         <t:value-of select="schema-element(elem-anyURI)"/>
      </out>
   </t:template>
</t:transform>
