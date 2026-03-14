<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Mixing versions in two stylesheet modules -->
  
  <xsl:include href="backwards-001.xsl"/>

  <xsl:template name="two">
    <out-two>
      <xsl:call-template name="main"/>
      <two><xsl:value-of select="1 to 5"/></two>
    </out-two>
  </xsl:template>

</xsl:stylesheet>
