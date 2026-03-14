<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test a predicate that uses the "ancestor" axes. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select = "//child2[ancestor::element2]"/>
  </out>
</xsl:template>
</xsl:stylesheet>
