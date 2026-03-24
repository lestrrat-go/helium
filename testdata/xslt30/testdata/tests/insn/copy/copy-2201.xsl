<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copy-of?>
    <!-- Purpose: Use key() to try to create a set of nodes in random order. -->

<xsl:key name="k" use="@origin" match="a" />

<xsl:template match="/">
  <out>
    <xsl:copy-of select="key('k','Albany')"/>
  </out>
</xsl:template>

</xsl:stylesheet>