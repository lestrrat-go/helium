<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copy-of?>   
    <!-- Purpose: Test for copy-of with '*' wildcard pattern -->
  <!-- This test also checks handling of processing instructions by copy-of. -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
