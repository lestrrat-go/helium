<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Test for select that comes up empty.-->

<xsl:template match="/">
  <out>
    <xsl:for-each select="doc//Q">
      <xsl:text>Found a Q!</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
