<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.schemanodetest.example.com/ns/subgroup"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="my xs"
                version="2.0">
<!-- Purpose: Test that the nilled property of a candidate node is 'false' if it passes the schema-element($name) 
				test, where $name is the name of a top level element declaration which is not nillable.-->

   <xslt:import-schema namespace="http://www.schemanodetest.example.com/ns/subgroup"
                       schema-location="subgroupTypeNodeTest.xsd"/>

   <xslt:template match="/my:doc">
	     <out>
         <xslt:apply-templates select="my:elem-URI"/>
      </out>
   </xslt:template>

   <xslt:template match="my:elem-URI">
	     <xslt:value-of select="schema-element(my:elem-anyURI)"/>
	     <xslt:value-of select="nilled(my:elem-anyURI)"/>
   </xslt:template>
</xslt:transform>
