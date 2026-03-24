<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="version" static="yes" select="'0.0'"/>
  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="$version"/></out>
  </xsl:template>
</xsl:stylesheet>
