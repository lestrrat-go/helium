<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- FileName: PREDICATE07 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of implied conversion to number for an inequality test
       in backwards compatibility mode. -->

<xsl:template match="doc">
  <out>
  <xsl:for-each select="a[0 &lt; true()]">
    <!-- Always true; true converts to 1. -->
    <xsl:value-of select="."/>
  </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
