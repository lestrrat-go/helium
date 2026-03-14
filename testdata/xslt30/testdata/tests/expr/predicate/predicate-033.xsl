<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: predicate40 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
  <!-- LocationPath002 in NIST suite -->
  <!-- Purpose: Test a predicate that uses the "ancestor" axis. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select = "//child2[ancestor::element2]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
