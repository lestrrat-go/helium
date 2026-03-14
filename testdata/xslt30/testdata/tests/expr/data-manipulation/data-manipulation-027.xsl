<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#variables?>
  <!-- Purpose: Test an "xsl:variable" using the "ancestor" 
       axes as part of the "select" attribute. -->

<xsl:variable name="variable1" select="//child1[ancestor::element1]"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$variable1"/>
 </out>
</xsl:template>

</xsl:stylesheet>
