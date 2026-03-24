<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Use xsl:copy with a single attribute set. -->

  <xsl:template match="test">
    <out>
      <xsl:copy use-attribute-sets="set1"/>
    </out>
  </xsl:template>

  <xsl:attribute-set name="set1">
    <xsl:attribute name="format">bold</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="text()"/>

</xsl:stylesheet>
