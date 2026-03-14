<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate' function with a param reference 
       for all its arguments.-->

<xsl:param name="param1">translate 1234 param</xsl:param>
<xsl:param name="param2">1234</xsl:param>
<xsl:param name="param3">with</xsl:param>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate($param1,$param2,$param3)"/>
 </out>
</xsl:template>
</xsl:stylesheet>