<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring?>
  <!-- Purpose: Test of 'substring' function with a param reference 
       as its first argument. Third argument is missing.-->

<xsl:param name="param1">do no show thissubstring with param</xsl:param>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring($param1,16)"/>
 </out>
</xsl:template>
</xsl:stylesheet>