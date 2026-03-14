<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test of 'generate-id()' - ensure same node generates same ID. -->

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="generate-id(a)=generate-id(a)"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
