<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- author MHK -->
  <!-- Purpose: Test for xsl:processing-instruction with a select attribute -->
  <?spec xslt#creating-processing-instructions?>

<xsl:template match="/">
  <out>
    <xsl:processing-instruction name="{doc}" select="1 to 3"/>
  </out>
</xsl:template>

</xsl:stylesheet>