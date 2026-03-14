<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#parameters?>
  <!-- Purpose: Test an "xsl:param" using a complex
       expression as part of the "select" attribute. -->

<xsl:param name="param1" select="doc/element1[((((((2*10)-4)+9) div 5) mod 3)+1)]"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$param1"/>
 </out>
</xsl:template>

</xsl:stylesheet>
