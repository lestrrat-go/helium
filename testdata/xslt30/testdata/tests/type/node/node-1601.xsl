<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test that 'namespace::node()' selects all namespaces. -->

<xsl:template match="doc">
  <xsl:element name="NSlist">
    <xsl:for-each select="namespace::node()">
      <xsl:attribute name="{name(.)}"><xsl:value-of select="."/></xsl:attribute>
    </xsl:for-each>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
