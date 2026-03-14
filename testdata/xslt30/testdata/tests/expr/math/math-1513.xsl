<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-paren-expressions?>
    <!-- Purpose: Test of nesting of parentheses. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="((((((n3+5)*(3)+(((n2)+2)*(n1 - 6)))-(n4 - n2))+(-(4-6)))))"/>
  </out>
</xsl:template>

</xsl:stylesheet>
