<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with schema-attribute($name) where $name is a top level attribute declaration 
  				in a schema with no target namespace.-->

   <t:import-schema schema-location="builtinTypeNodeTest.xsd"/>

   <t:template match="/doc">
	     <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="elem-ALL-attr-Types">
	     <t:value-of select="schema-attribute(attr-anyURI)"/>
   </t:template>
</t:transform>
