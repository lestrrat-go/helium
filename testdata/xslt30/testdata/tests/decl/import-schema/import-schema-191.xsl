<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test that an error is raised if the processor is set to Basic 
  				Processor and the stylesheet includes an xsl:import-schema.-->
 
   <xsl:import-schema namespace="http://ns.example.com/sch002"
                      schema-location="schema002.xsd"/>

   <xsl:template match="/doc">
      <out>test</out>
   </xsl:template>



</xsl:stylesheet>
