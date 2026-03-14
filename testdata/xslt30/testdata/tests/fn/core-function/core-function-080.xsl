<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Section: 4.4 Number Functions -->
  <!-- Purpose: Test of 'round' function with NAN as its
       its argument. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="round(number('NaN'))"/>
 </out>
</xsl:template>
</xsl:stylesheet>