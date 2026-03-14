<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test a matching patttern with a complex
                expression. -->
<xsl:template match = "/">
  <xsl:apply-templates select="doc/element1[(((((2*10)-4)+9) div 5) mod 3)]/child1[last()]"/>
</xsl:template>

<xsl:template match="doc/element1[(((((2*10)-4)+9) div 5) mod 3)]/child1[last()]">
  <out>
    <xsl:value-of select = "."/>
  </out>
</xsl:template>
</xsl:stylesheet>
