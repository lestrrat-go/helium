<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Purpose: Test of 'round' function with "ceiling"
       function as its argument. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="round(ceiling(3.2))"/>
 </out>
</xsl:template>
</xsl:stylesheet>