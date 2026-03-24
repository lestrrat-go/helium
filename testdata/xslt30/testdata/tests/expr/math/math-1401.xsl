<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#value-of?>
    <!-- Purpose: Test what value-of does to a large number. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="9876543210"/>
  </out>
</xsl:template>

</xsl:stylesheet>
