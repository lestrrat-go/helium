<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE27 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of implied conversion of node-set to number with a != relation. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select = "a[4 != following-sibling::*]">
      <xsl:value-of select="."/>
      <!-- True only when there is a following sibling that does not contain a 4. -->
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
