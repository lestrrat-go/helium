<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>
    <!-- Purpose: Test of current() function - just select it. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="current()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
