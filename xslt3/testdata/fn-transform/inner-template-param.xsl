<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template">
    <xsl:param name="color" select="'none'"/>
    <out><xsl:value-of select="$color"/></out>
  </xsl:template>
</xsl:stylesheet>
