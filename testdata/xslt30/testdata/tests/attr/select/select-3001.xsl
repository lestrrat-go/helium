<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: NCName after [ must not be treated as operator -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="a[div=9]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
