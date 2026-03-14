<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: name after * should not be treated as operator -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="' 6 '*div"/>
  </out>
</xsl:template>

</xsl:stylesheet>
