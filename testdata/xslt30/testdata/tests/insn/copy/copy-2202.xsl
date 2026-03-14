<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy40 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 Using Values of Variables & Parameters with xsl:copy-of. -->
    <!-- Purpose: Use key() to get nodes from various places, then copy and mark. -->

<xsl:key name="k" use="@origin" match="a" />

<xsl:template match="/">
  <out>
    <xsl:for-each select="key('k','Albany')">
      <xsl:copy>
        <xsl:copy-of select="@level"/>
        <xsl:attribute name="data"><!-- Mark copied 'a' nodes with their data -->
          <xsl:value-of select="text()"/>
        </xsl:attribute>
      </xsl:copy><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>