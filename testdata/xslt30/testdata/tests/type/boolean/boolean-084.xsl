<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#id-logical-expressions?>
    <!-- Purpose: Test that "and" doesn't bother with right operand if left is false 
       when in backwards compatibility mode -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="false() and 1 div 0"/>
  </out>
</xsl:template>

</xsl:stylesheet>
