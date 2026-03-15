<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test a predicate that uses elaborate complex
       expressions -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select = "element1[(((((2*10)-4)+9) div 5) mod 3 )]"/>
  </out>
</xsl:template>
</xsl:stylesheet>
