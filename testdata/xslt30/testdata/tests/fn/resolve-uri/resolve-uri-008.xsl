<?xml version='1.0'?>
<?spec xslt#locating-modules?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Section: 3.2 Base URI (Stylesheet import/Inclusion) -->
    <!-- Purpose: This test verifies correct URI resolution with relative URI's. --> 
  <!-- Case: include, include, include, include -->

<xsl:include href="level1/level1_include4.xsl"/>

<xsl:template match="root-tag">
 <out>
Case of: include, include, include, include
  Including level1_include4.xsl
  <xsl:apply-templates/>
 </out>
</xsl:template>

</xsl:stylesheet>