<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: test union operator where union yields empty set. -->
  <!-- Expression012 in NIST suite -->

<xsl:template match = "doc">
  <out><xsl:text>
</xsl:text>
    <xsl:apply-templates select = "//noChild1|//noChild2"/>
  </out>
</xsl:template>

<xsl:template match = "*">
  <xsl:value-of select = "."/><xsl:text>
    </xsl:text>
</xsl:template>

</xsl:stylesheet>