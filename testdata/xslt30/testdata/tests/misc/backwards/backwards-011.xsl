<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: effect of BC on xsl:number -->

  <xsl:template name="main">
    <out>
      <one xsl:version="1.0"><xsl:number value="1 to 5"/></one>
      <two xsl:version="2.0"><xsl:number value="1 to 5"/></two>
    </out>
  </xsl:template>

</xsl:stylesheet>
