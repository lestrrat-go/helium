<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: test that a statically declared variable is visible in a subsequently-included module. -->

  <xsl:template name="action" use-when="$inc">
    <ok/>
  </xsl:template>
  
  <xsl:variable name="oink" select="true()" static="yes"/>

</xsl:stylesheet>