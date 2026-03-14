<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: * after [ must not be treated as operator -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="a[*=9]">
      <xsl:value-of select="@s"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
