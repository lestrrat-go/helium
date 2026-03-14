<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Mixing versions in one stylesheet module, nested override -->

  <xsl:template name="main" version="1.0">
    <out xsl:version="2.0">
      <one xsl:version="1.0"><xsl:value-of select="1 to 5"/></one>
      <two><xsl:value-of select="1 to 5"/></two>
    </out>
  </xsl:template>

</xsl:stylesheet>
