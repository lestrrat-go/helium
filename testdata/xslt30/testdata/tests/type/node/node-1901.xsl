<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- author MHK -->
  <!-- Purpose: Test for xsl:attribute producing a sequence -->
  <?spec xslt#creating-attributes?>

<xsl:template match="/">
  <out>
    <xsl:attribute name="v" select="1 to 10"/>
  </out>
</xsl:template>

</xsl:stylesheet>