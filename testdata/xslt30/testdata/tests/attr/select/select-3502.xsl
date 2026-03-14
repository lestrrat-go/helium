<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: * after operator should be treated as path -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="5.+*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
