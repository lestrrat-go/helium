<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on arithmetic expressions -->

  <xsl:template name="main">
    <out><xsl:value-of select="1 + (6 to 10)"/></out>
  </xsl:template>


</xsl:stylesheet>
