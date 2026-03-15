<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-before?>
  <!-- Purpose: Test of 'substring-before' function with a param reference 
       for all its arguments.-->

<xsl:param name="param1">substring-before with paramcut this</xsl:param>
<xsl:param name="param2">cut this</xsl:param>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring-before($param1,$param2)"/>
 </out>
</xsl:template>
</xsl:stylesheet>