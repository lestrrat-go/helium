<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test a predicate that uses the "child" axes. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="child1[child::child2]"/>
  </out>
</xsl:template>
</xsl:stylesheet>
