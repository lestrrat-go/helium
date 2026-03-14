<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '=' operator with one number with leading zero, one not. Reverse order of boolean14.-->
  <!-- Error: cannot compare a number to a string -->
  <?same-as-1.0 no?>

<xsl:template match="/">
  <out>
    <xsl:value-of select="'001' = 1"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
