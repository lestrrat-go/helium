<?xml version="1.0" ?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <xsl:import href="attribute-set-1804a.xsl"/>
 <xsl:import href="attribute-set-1804b.xsl"/>


 <?spec xslt#attribute-sets?>
 <!-- Purpose: Basic test of import precedence based on Richard Titmuss's test
     with attribute sets. Here the imported attribute sets have additional non-
     conflicting attributes as well.  -->


 <xsl:template match="/">
  <out>
   <foo xsl:use-attribute-sets="bar"/>
  </out>
 </xsl:template>

</xsl:stylesheet>
