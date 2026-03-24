<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#id-logical-expressions?>
    <!-- Purpose: Test that "or" doesn't evaluate right operand if left is true 
       when in backwards compatibility mode -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="true() or 1 div 0"/>
  </out>
</xsl:template>

</xsl:stylesheet>
