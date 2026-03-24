<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: test for a conflict between two values of static variables. -->
     
  <xsl:import href="use-when-0134a.xsl"/>
  
    <xsl:variable name="inc" select="true()" static="yes"/>  

    
  <xsl:template name="main" use-when="$inc">
    <xsl:call-template name="action"/>
  </xsl:template>

</xsl:stylesheet>