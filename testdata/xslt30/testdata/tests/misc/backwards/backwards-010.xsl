<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: effect of BC on AVTs -->

  <xsl:template name="main">
    <out>
      <one xsl:version="1.0" select="{1 to 5}"/>
      <two xsl:version="2.0" select="{1 to 5}"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
