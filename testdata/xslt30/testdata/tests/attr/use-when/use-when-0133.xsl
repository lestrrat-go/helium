<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: test that a statically declared variable is visible in a subsequently-included module,
     and variables declared in the included module are visible in the including module after the xsl:include. -->
  
  <xsl:variable name="inc" select="true()" static="yes"/> 
  <xsl:include href="use-when-0133a.xsl"/> 
  
  <xsl:template name="main" use-when="$oink">
    <xsl:call-template name="action"/>
  </xsl:template>

</xsl:stylesheet>