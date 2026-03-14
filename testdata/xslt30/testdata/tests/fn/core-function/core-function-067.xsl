<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-floor?>
  <!-- Purpose: Test of 'floor' function with a complex expression
       as its argument.-->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="floor((((((2*10)-4)+9) div 5) mod 2))"/>
 </out>
</xsl:template>
</xsl:stylesheet>