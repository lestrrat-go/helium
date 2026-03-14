<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for-each inside xsl:element -->

<xsl:template match="docs">
  <out>
    <xsl:element name="all">
      <xsl:for-each select="a">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
