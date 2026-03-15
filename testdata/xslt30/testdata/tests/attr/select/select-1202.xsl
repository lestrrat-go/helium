<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: NCName followed by :: must be recognized as an AxisName,
      even if there is intervening whitespace. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="child :: sub">
      <xsl:value-of select="./text()[1]"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
