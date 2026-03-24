<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#parameters?>
  <!-- Purpose: Test an "xsl:param" using another param
       as part of the "select" attribute. -->

<xsl:param name="param1" select="doc/element1"/>
<xsl:param name="param2" select="$param1"/>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "$param2"/>
 </out>
</xsl:template>

</xsl:stylesheet>
