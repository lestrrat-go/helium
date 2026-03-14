<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: NCName followed by ( must be recognized as a NodeType
      or FunctionName, even if there is intervening whitespace. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="string-length ()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
