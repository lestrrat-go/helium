<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on xsl:call-template (no error raised for unwanted parameter) -->
  
  <xsl:include href="backwards-001.xsl"/>

  <xsl:template name="two">
    <out>
      <xsl:call-template name="main">
        <xsl:with-param name="unwanted" select="2"/>
      </xsl:call-template>
    </out>
  </xsl:template>

</xsl:stylesheet>
