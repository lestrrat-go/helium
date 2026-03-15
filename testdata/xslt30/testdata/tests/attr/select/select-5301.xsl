<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: test union operator used three times in an expression -->
  <!-- Expression023 in NIST suite -->

<xsl:template match = "doc">
  <out>
    <xsl:for-each select = "//child5|//child2|//child4">
      <xsl:value-of select="."/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>