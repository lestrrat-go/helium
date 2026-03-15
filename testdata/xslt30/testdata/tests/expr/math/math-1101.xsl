<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-number?>
    <!-- Purpose: Test that number() with no argument applies to context node. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="number()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
