<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: Basic test of backwards compatibility mode. -->

  <xsl:template name="main">
    <out>
      <xsl:value-of select="1 to 5"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
