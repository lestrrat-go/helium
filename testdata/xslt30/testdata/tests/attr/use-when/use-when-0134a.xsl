<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: test that a statically declared variable is visible in a subsequently-imported module. -->

  <xsl:template name="action">
    <ok/>
  </xsl:template>
  
  <xsl:variable name="inc" select="false()" static="yes"/>

</xsl:stylesheet>