<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate()' function with a variable reference 
       for all three arguments.-->

<xsl:variable name="variable1">translate 1234 variable</xsl:variable>
<xsl:variable name="variable2">1234</xsl:variable>
<xsl:variable name="variable3">with</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate($variable1,$variable2,$variable3)"/>
 </out>
</xsl:template>
</xsl:stylesheet>