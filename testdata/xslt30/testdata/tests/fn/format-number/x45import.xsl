<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"> 

  <!-- FileName: x45import -->
  <!-- Purpose: Imported by NumberFormat45 -->

<xsl:decimal-format decimal-separator=',' grouping-separator='.'/>

<xsl:template name="sub">
  <sub>
    <xsl:value-of select="format-number(12345.67,'000.000,###')"/>
    <xsl:text>, </xsl:text>
    <xsl:value-of select="format-number(-98765.4321,'##0.000,000')"/>
  </sub>
</xsl:template>

</xsl:stylesheet>
