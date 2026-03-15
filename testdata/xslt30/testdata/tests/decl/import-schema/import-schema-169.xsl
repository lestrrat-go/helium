<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test where a schema is imported in the primary stylesheet  
				but datatypes are used in an included stylesheet.-->
 
   <xsl:include href="import-schema-168a.xsl"/>
   <xsl:import-schema namespace="http://ns.example.com/sch002"
                      schema-location="schema002.xsd"/>


</xsl:stylesheet>
