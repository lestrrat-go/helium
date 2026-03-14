<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: union of two absolute (//) paths -->
  <!-- Expression022 in NIST suite -->

<xsl:template match = "doc">
  <out>
    <xsl:apply-templates select = "//child1|//child2"/>
  </out>
</xsl:template>

<xsl:template match = "*">
  <xsl:value-of select = "."/>
</xsl:template>

</xsl:stylesheet>

