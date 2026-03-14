<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test != on two node-sets that have one node in common. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="j[@l='12'] != j[@w='45']"/>
  </out>
</xsl:template>

</xsl:stylesheet>