<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE26 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
    <!-- Section: 3.4 -->
  <!-- Purpose: Test of implied conversion of node-set to number, ensure = is symmetric. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="a[following-sibling::*=3]">
      <!-- True only when the node containing 3 is one of the following siblings. -->
      <xsl:value-of select="."/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
