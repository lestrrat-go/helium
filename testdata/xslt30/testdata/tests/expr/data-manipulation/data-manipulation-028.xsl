<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#parameters?>
  <!-- Purpose: Test an "xsl:param" using the "ancestor" 
       axes as part of the "select" attribute. -->

<xsl:param name="param1" select="//child1[ancestor::element1]"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$param1"/>
 </out>
</xsl:template>

</xsl:stylesheet>
