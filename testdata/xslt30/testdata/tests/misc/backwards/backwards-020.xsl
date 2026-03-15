<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on function call argument conversions -->

  <xsl:template name="main">
    <out><xsl:value-of select="substring(current-date(), 1, 2)"/></out>
  </xsl:template>


</xsl:stylesheet>
