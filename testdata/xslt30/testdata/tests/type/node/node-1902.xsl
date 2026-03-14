<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- author MHK -->
  <!-- Purpose: Test for xsl:comment with a select attribute -->
  <?spec xslt#creating-comments?>

<xsl:template match="/">
  <out>
    <xsl:comment select="doc, doc, doc"/>
  </out>
</xsl:template>

</xsl:stylesheet>