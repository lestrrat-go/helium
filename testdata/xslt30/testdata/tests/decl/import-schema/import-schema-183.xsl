<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test of xsl:import-schema without attributes and with empty content. Do not use any 
				non-built-in data types to prevent error since GrammarPool is empty by default.-->
 
   <xsl:import-schema/>

   <xsl:template match="/doc">
      <out>test</out>
   </xsl:template>



</xsl:stylesheet>
