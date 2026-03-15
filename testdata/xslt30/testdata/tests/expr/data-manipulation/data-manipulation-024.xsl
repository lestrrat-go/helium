<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#parameters?>
  <!-- Purpose: Test an "xsl:variable" using a function
       as part of the "select" attribute. -->

<xsl:param name="variable1" select="doc/element1[last()]"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$variable1"/>
 </out>
</xsl:template>

</xsl:stylesheet>
