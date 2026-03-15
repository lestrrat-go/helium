<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- Purpose: test IEEE 754 comparisons involving sign of zero.
       (test of "not equal to" of +0 and -0) -->
  <!--  -->

<xsl:template match = "doc">
 <out>
   <xsl:value-of select = "(-0 != 0)"/>
 </out>
</xsl:template>

</xsl:stylesheet>
