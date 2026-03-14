<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
 
 <!-- derivative of func39 to test progressive reading of a closure -->
 
 <xsl:variable name="root" select="/"/>
 
 <xsl:template match="/">
  <xsl:variable name="x" select="for $i in 1 to 10 return $i*$i"/>
  <out>
  <xsl:for-each select="1 to 10">
    <x value="{$x[current()]}"/>
  </xsl:for-each>
  </out>
 </xsl:template>


</xsl:stylesheet>
