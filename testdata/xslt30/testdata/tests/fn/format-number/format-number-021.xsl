<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT22 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test import of a decimal-format. Three formats should not conflict. -->

<xsl:import href="periodgroup.xsl"/>

<xsl:decimal-format name="myminus" minus-sign='_' />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-26931.4,'###.###,###','periodgroup')"/>
    <xsl:text>  </xsl:text>
    <xsl:value-of select="format-number(-42857.1,'###,###.###','myminus')"/>
    <xsl:text>  </xsl:text>
    <xsl:value-of select="format-number(-73816.9,'###,###.###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
