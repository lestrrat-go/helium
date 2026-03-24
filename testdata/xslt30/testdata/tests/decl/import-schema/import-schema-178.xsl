<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test with xsl:import-schema where no schema is found for the namespace 
  				specified. No components are used from this namespace. -->
 
   <xsl:import-schema namespace="http://ns.example.com/nonexistent"/>

   <xsl:template match="/doc">
      <out>test</out>
   </xsl:template>



</xsl:stylesheet>
