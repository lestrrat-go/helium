<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: xsl:value-of with separator, version is irrelevant -->

  <xsl:template name="main">
    <out>
      <one xsl:version="1.0"><xsl:value-of select="1 to 5" separator=","/></one>
      <two xsl:version="2.0"><xsl:value-of select="1 to 5" separator=","/></two>
    </out>
  </xsl:template>

</xsl:stylesheet>
