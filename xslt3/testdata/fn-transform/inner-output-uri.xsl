<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="current-output-uri()"/></out>
  </xsl:template>
</xsl:stylesheet>
