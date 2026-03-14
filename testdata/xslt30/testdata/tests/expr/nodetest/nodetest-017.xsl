<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration. The type of the candidate 
  				node is derived directly from the type of the declaration in the schema.The candidate node is validated 
  				against a local element declaration with the same expanded-QName as the top-level element declaration.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                    schema-location="subgroupTypeNodeTest.xsd"/>

   <t:template match="/my:doc">
	     <out>
         <t:apply-templates select="my:elem-URI"/>
         <t:apply-templates select="my:elem-INT"/>
      </out>
   </t:template>

   <t:template match="my:elem-URI">
	     <E1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-element(my:elem-anyURI)"/>
      </E1>
   </t:template>

   <t:template match="my:elem-INT">
	     <E2 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-element(my:elem-anyURI)"/>
      </E2>
   </t:template>
</t:transform>
