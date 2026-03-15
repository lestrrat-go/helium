<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>
    <!-- Purpose: Show that current() produces a node-set. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="current()">
      <xsl:value-of select="."/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
