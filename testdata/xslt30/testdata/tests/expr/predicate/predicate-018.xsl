<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE11 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of comparison of 2 node-sets. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="a[following-sibling::*=descendant::*]"/>
    <!-- True only when the node has both a descendant and a following sibling containing "target". -->
  </out>
</xsl:template>

</xsl:stylesheet>
