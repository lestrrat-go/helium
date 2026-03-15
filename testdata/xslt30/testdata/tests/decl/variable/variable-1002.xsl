<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for using xsl:copy-of with empty (null string) variable. -->


<xsl:template match="doc">
  <xsl:variable name="x"/>
  <out>
    <xsl:copy-of select="$x"/>
  </out>
</xsl:template>

</xsl:stylesheet>
