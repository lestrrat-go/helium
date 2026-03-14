<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: predicate41 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
  <!-- LocationPath003 in NIST suite -->
  <!-- Purpose: Test a predicate that uses the "ancestor-or-self" axis. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select = "//child2[ancestor-or-self::element2]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
