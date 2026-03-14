<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"> 

  <!-- FileName: NumberFormat44 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test whether a decimal-format declaration in an import is visible here. -->
  <!-- Runs with version="1.0" -->

<xsl:import href="x43import.xsl"/>

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <main>
      <xsl:value-of select="format-number('NaN','###','decimal3')"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="format-number(-13.2,'###.0','decimal3')"/>
    </main>
    <xsl:text>&#10;</xsl:text>
    <xsl:call-template name="sub"/>
  </out>
</xsl:template>

</xsl:stylesheet>
