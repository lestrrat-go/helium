<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Purpose: Test of 'round' function with a variable
       reference as its argument. -->

<xsl:variable name="variable1">2.3</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="round($variable1)"/>
 </out>
</xsl:template>
</xsl:stylesheet>