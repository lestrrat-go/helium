<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#local-variables?>
  <!-- Purpose: test union operator using variables. -->
  <!-- Expression013 in NIST suite -->

<xsl:template match = "doc">
  <xsl:variable name = "var1" select = "//child1"></xsl:variable>
  <xsl:variable name = "var2" select = "//child2"></xsl:variable>
  <out><xsl:text>
</xsl:text>
    <xsl:apply-templates select = "$var1|$var2"/>
  </out>
</xsl:template>

<xsl:template match = "*">
  <xsl:value-of select = "."/><xsl:text>
    </xsl:text>
</xsl:template>

</xsl:stylesheet>