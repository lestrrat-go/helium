<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Use xsl:element with for-each inside xsl:attribute -->

<xsl:template match="docs">
  <xsl:element name="out">
    <xsl:attribute name="test1">
      <xsl:for-each select="a">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:attribute>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>