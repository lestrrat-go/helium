<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: * after (ambiguous) operator must not be treated as operator -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="25-*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
