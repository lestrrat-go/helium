<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE12 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of boolean or. -->

<xsl:template match="doc">
  <out>

  <xsl:for-each select="a[('target'=descendant::*) or @squish]">
    <xsl:value-of select="."/>
    <!-- True only when the node has either a squish attribute or a descendant containing "target". -->
  </xsl:for-each>

  </out>
</xsl:template>

</xsl:stylesheet>
