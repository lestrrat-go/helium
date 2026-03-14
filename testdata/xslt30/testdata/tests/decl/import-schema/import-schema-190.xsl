<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:foo="http://ns.example.com/redefine"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="foo xs"
                version="2.0">

  <!-- Purpose: Test to show that importing a schema implicitly imports other schemas 
  				that are referenced from that schema using <xs:redefine>.-->
 
   <xsl:import-schema namespace="http://ns.example.com/redefine"
                      schema-location="schema190a.xsd"/>

   <xsl:variable name="var" as="element()">
	     <foo:root xsl:type="foo:rootType">
         <foo:elem1>1</foo:elem1>
         <foo:elem2>-1</foo:elem2>
         <foo:elem3>elem3</foo:elem3>
         <foo:elem4>elem4</foo:elem4>
      </foo:root>
   </xsl:variable>

   <xsl:template match="/doc">
	     <out>
         <xsl:value-of select="$var instance of element(*, foo:rootType)"/>
         <xsl:value-of select="$var instance of schema-element(foo:root)"/>
         <xsl:value-of select="$var/*"/>
      </out>
   </xsl:template>



</xsl:stylesheet>
