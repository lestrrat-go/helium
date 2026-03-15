<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-starts-with?>
  <!-- Purpose: Test of 'starts-with' function with a param reference 
       as its second argument.-->

<xsl:param name="param1">This</xsl:param>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="starts-with('This is a string',$param1)"/>
 </out>
</xsl:template>
</xsl:stylesheet>