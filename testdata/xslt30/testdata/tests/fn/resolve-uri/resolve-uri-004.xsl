<?xml version='1.0'?>
<?spec xslt#locating-modules?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


    <!-- Purpose: This test verifies correct URI resolution with relative URI's. --> 
  <!-- Case: import, include, include, import -->

<xsl:import href="level1/level1_include2.xsl"/>

<xsl:template match="root-tag">
 <out>
Case of: import, include, include, import
  importing level1_include2.xsl
  <xsl:apply-templates/>
 </out>
</xsl:template>

</xsl:stylesheet>