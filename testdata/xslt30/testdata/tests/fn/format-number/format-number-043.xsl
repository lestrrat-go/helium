<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NumberFormat46 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test whether a default decimal-format defined in an include is applied here. -->

<xsl:include href="x45import.xsl"/>

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <one>
      <xsl:value-of select="format-number(12345.67,'000.000,###')"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="format-number(-98765.4321,'##0.000,000')"/>
    </one>
    <xsl:text>&#10;</xsl:text>
    <xsl:call-template name="sub"/>
  </out>
</xsl:template>

</xsl:stylesheet>
