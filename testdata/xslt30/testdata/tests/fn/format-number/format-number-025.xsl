<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: NUMBERFORMAT27 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of format-number minus-sign behavior on positive numbers. -->

<xsl:decimal-format name="myminus" minus-sign='_' />

<xsl:template match="doc">
  <out>All three should be the same...<xsl:text>
</xsl:text><xsl:value-of select="format-number(2392.14*36.58,'000,000.000000','myminus')"/><xsl:text>
</xsl:text><xsl:value-of select="format-number(2392.14*36.58,'000,000.000000;###,###.000###')"/><xsl:text>
</xsl:text><xsl:value-of select="format-number(2392.14*36.58,'000,000.000000;###,###.000###','myminus')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
