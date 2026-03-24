<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position03 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() function anded with attribute test. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="*[@test and position()=8]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
