<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on general comparisons -->

  <xsl:template name="main">
    <out><xsl:value-of select="1 = true()"/></out>
  </xsl:template>


</xsl:stylesheet>
