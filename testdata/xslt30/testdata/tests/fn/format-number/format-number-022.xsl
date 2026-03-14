<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT23 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of include that does an import, providing two named decimal-formats. Three formats should not conflict. -->

<xsl:include href="newminus.xsl"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-26931.4,'###.###,###','periodgroup')"/>
    <xsl:text>  </xsl:text>
    <xsl:value-of select="format-number(-42857.1,'###,###.###','newminus')"/>
    <xsl:text>  </xsl:text>
    <xsl:value-of select="format-number(-73816.9,'###,###.###')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
