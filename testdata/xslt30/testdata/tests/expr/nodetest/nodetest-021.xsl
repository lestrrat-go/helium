<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="my xs"
                version="2.0">
<!-- Purpose: Show that if the name of a node in the source file complies to the schema-element($name) test but its type 
  				(different than xs:untyped) does not, the test will fail. The candidate node is validated against a local 
  				element declaration with the same expanded-QName as the top-level element declaration.-->

   <xslt:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                       schema-location="subgroupTypeNodeTest.xsd"/>

   <xslt:template match="/my:doc">
	     <out>
         <xslt:apply-templates select="my:elem-INT"/>
      </out>
   </xslt:template>

   <xslt:template match="my:elem-INT">
	     <xslt:value-of select="my:elem-anyURI instance of element(*, xs:integer)"/>
	     <xslt:value-of select="my:elem-anyURI instance of element(*, my:derivedURI)"/>
	     <xslt:value-of select="schema-element(my:elem-anyURI)"/>
   </xslt:template>
</xslt:transform>
