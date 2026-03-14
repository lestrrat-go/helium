<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: predicate46 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
  <!-- LocationPath008 in NIST suite -->
  <!-- Purpose: Test a predicate that uses math functions. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select = "element1[(((((2*10)-4)+9) div 5) mod floor(3))]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
