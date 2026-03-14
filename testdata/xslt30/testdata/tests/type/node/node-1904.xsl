<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- author MHK -->
  <!-- Purpose: Test for xsl:namespace with a select attribute -->
  <?spec xslt#creating-namespace-nodes?>

<xsl:template match="/">
  <out>
    <xsl:namespace name="prefix" select="'http://uri.com/'"/>
  </out>
</xsl:template>

</xsl:stylesheet>