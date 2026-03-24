<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Purpose: Test of 'round' function with an argument
       between 0 and -0.5 (inclusive). -->
       <?same-as-1.0 no?>

  <!-- the value returned is negative zero, but it's displayed as "0" - MHK -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="round(-0.25e0)"/>
 </out>
</xsl:template>
</xsl:stylesheet>