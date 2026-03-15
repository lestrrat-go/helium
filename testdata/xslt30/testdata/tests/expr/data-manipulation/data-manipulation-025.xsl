<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#variables?>
  <!-- Purpose: Test an "xsl:variable" using a complex
       expression as part of the "select" attribute. -->

<xsl:param name="variable1" select="doc/element1[((((((2*10)-4)+9) div 5) mod 3)+1)]"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$variable1"/>
 </out>
</xsl:template>

</xsl:stylesheet>
