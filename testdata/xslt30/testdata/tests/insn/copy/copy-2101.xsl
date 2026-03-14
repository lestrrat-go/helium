<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copy-of?>
    <!-- Purpose: current() should work in copy-of. -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="."/>
    <xsl:text>&#10;</xsl:text>
    <xsl:copy-of select="current()"/>
  </out>
</xsl:template>

</xsl:stylesheet>