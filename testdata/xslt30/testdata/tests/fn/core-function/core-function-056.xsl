<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate' function with a param reference 
       as its first argument.-->

<xsl:param name="param1">translate 1234 param</xsl:param>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate($param1,'1234','with')"/>
 </out>
</xsl:template>
</xsl:stylesheet>