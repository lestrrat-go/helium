<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:my="http://example.com/my">
  <xsl:param name="my:greeting" select="'default'"/>
  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="$my:greeting"/></out>
  </xsl:template>
</xsl:stylesheet>
