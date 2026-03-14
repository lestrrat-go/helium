<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: numberformat934err -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test changing both digit and zero-digit in format string. 
       Fails XTSE1295 because the replacement zero-digit character is not actually a digit -->

<xsl:decimal-format digit="!" zero-digit="a" />

<xsl:template name="main">
  <out>
    <xsl:value-of select="format-number(4030201.0506,'#!!!,!!!,aaa.aaaaaa0')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
