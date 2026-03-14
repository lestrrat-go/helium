<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-floor?>
  <!-- Purpose: Test of 'floor' function with a "param"
       reference as its argument.-->

<xsl:param name="param1">4.1</xsl:param>
<xsl:template match="doc">
 <out>
   <xsl:value-of select="floor($param1)"/>
 </out>
</xsl:template>
</xsl:stylesheet>