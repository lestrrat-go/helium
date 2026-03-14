<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: NCName after :: must be treated as part of path -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="attribute :: div"/>
  </out>
</xsl:template>

</xsl:stylesheet>
