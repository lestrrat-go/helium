<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"> 

  <!-- FileName: NumberFormat43 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Have two decimal-format elements with identical names, one in an import.
    This is allowed as long as all attributes are identical (including defaults). -->
  <!-- Runs with version="1.0" -->

<xsl:import href="x43import.xsl"/>

<xsl:decimal-format name="decimal3" NaN='not a number' decimal-separator='.'/>

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <one>
      <xsl:value-of select="format-number('NaN','###','decimal3')"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="format-number(-13.2,'###.0','decimal3')"/>
    </one>
    <xsl:text>&#10;</xsl:text>
    <xsl:call-template name="sub"/>
  </out>
</xsl:template>

</xsl:stylesheet>
